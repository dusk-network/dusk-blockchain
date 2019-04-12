package reduction

import (
	"bytes"
	"encoding/hex"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	collector struct {
		*consensus.StepEventCollector
		collectedVotesChan chan []wire.Event
		queue              *consensus.EventQueue
		lock               sync.RWMutex
		reducer            *reducer
		ctx                *context

		publisher wire.EventPublisher
	}

	// Broker is the message broker for the reduction process.
	broker struct {
		eventBus  *wire.EventBus
		collector *collector
		// utility context to group interfaces and channels to be passed around
		ctx *context

		// channels linked to subscribers
		roundUpdateChan <-chan uint64
		selectionChan   chan []byte

		// utility
		unMarshaller *unMarshaller

		outgoingReductionTopic string
		outgoingAgreementTopic string

		// TODO: review this after demo. used to restart phase after reduction
		generationTopic string
		reductionTopic  topics.Topic
	}
)

func newCollector(eventBroker wire.EventBroker, reductionTopic string, ctx *context) *collector {
	collector := &collector{
		StepEventCollector: consensus.NewStepEventCollector(),
		queue:              consensus.NewEventQueue(),
		collectedVotesChan: make(chan []wire.Event, 1),
		ctx:                ctx,
		publisher:          eventBroker,
	}

	go wire.NewTopicListener(eventBroker, collector, reductionTopic).Accept()
	go collector.onTimeout()
	return collector
}

func (c *collector) onTimeout() {
	for {
		<-c.ctx.timer.TimeoutChan
		c.Clear()
	}
}

func (c *collector) Clear() {
	c.StepEventCollector.Clear()
}

func (c *collector) Collect(buffer *bytes.Buffer) error {
	ev := c.ctx.handler.NewEvent()
	if err := c.ctx.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	header := &consensus.EventHeader{}
	c.ctx.handler.ExtractHeader(ev, header)
	if c.isRelevant(header.Round, header.Step) {
		if err := c.ctx.handler.Verify(ev); err != nil {
			return err
		}
		c.process(ev)
		return nil
	}

	if c.isEarly(header.Round, header.Step) {
		c.queue.PutEvent(header.Round, header.Step, ev)
	}

	return nil
}

func (c *collector) process(ev wire.Event) {
	b := new(bytes.Buffer)
	if err := c.ctx.handler.EmbedVoteHash(ev, b); err == nil {
		hash := hex.EncodeToString(b.Bytes())
		if !c.Contains(ev, hash) {
			c.repropagate(ev)
		}

		count := c.Store(ev, hash)
		if count > c.ctx.committee.Quorum() {
			votes := c.StepEventCollector.Map[hash]
			c.collectedVotesChan <- votes
			c.Clear()
		}
	}
}

func (c *collector) repropagate(ev wire.Event) {
	buf := new(bytes.Buffer)
	_ = c.ctx.handler.Marshal(buf, ev)
	message, _ := wire.AddTopic(buf, topics.BlockReduction)
	c.publisher.Publish(string(topics.Gossip), message)
}

func (c *collector) flushQueue() {
	queuedEvents := c.queue.GetEvents(c.ctx.state.Round(), c.ctx.state.Step())
	for _, event := range queuedEvents {
		c.process(event)
	}
}

func (c *collector) updateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "reducer",
		"round":   round,
	}).Debugln("Updating round")
	c.lock.Lock()
	if c.reducer != nil {
		c.reducer.stale = true
		c.reducer.end()
		c.reducer = nil
	}
	c.lock.Unlock()
	c.ctx.state.Update(round)

	c.queue.Clear(c.ctx.state.Round())
	c.Clear()
}

func (c *collector) isRelevant(round uint64, step uint8) bool {
	cmp := c.ctx.state.Cmp(round, step)
	if cmp == 0 && c.isReducing() {
		return true
	}
	log.WithFields(log.Fields{
		"process": "reducer",
		"cmp":     cmp,
		"reducer": c.isReducing(),
	}).Debugln("isRelevant mismatch")
	return false
}

func (c *collector) isReducing() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.reducer != nil
}

func (c *collector) isEarly(round uint64, step uint8) bool {
	return c.ctx.state.Cmp(round, step) <= 0
}

func (c *collector) startReduction() {
	log.Traceln("Starting Reduction")
	c.lock.Lock()
	c.reducer = newReducer(c.collectedVotesChan, c.ctx, c.publisher)
	c.lock.Unlock()

	go c.reducer.begin()
	c.flushQueue()
}

// newBroker will return a reduction broker.
func newBroker(eventBus *wire.EventBus, handler handler, selectionChannel chan []byte, committee committee.Committee, timeout time.Duration) *broker {

	ctx := newCtx(handler, committee, timeout)
	collector := newCollector(eventBus, string(topics.BlockReduction), ctx)

	roundChannel := consensus.InitRoundUpdate(eventBus)
	return &broker{
		eventBus:        eventBus,
		roundUpdateChan: roundChannel,
		unMarshaller:    newUnMarshaller(msg.VerifyEd25519Signature),
		ctx:             ctx,
		collector:       collector,

		selectionChan: selectionChannel,
	}
}

// Listen for incoming messages.
func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundUpdateChan:
			b.collector.updateRound(round)
		case hash := <-b.selectionChan:
			log.WithFields(log.Fields{
				"process": "reduction",
				"round":   b.collector.ctx.state.Round(),
				"hash":    hex.EncodeToString(hash),
			}).Debug("Got selection message")
			b.forwardSelection(hash)
		}
	}
}

func (b *broker) forwardSelection(hash []byte) {
	buf := bytes.NewBuffer(hash)
	vote, err := b.ctx.handler.MarshalHeader(buf, b.ctx.state)
	if err != nil {
		panic(err)
	}

	b.eventBus.Publish(b.outgoingReductionTopic, vote)
	b.collector.startReduction()
}
