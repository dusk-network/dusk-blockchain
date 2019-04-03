package reduction

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	collector struct {
		consensus.StepEventCollector
		collectedVotesChan chan []wire.Event
		queue              *consensus.EventQueue
		lock               sync.Mutex
		reducer            *reducer
		ctx                *context

		// TODO: review this, used to restart phase after reduction
		regenerationChannel chan bool

		// TODO: review re-propagation logic
		repropagationChannel chan *bytes.Buffer
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

func newCollector(eventBus *wire.EventBus, reductionTopic string, ctx *context) *collector {

	queue := consensus.NewEventQueue()
	collector := &collector{

		StepEventCollector:   consensus.StepEventCollector{},
		queue:                queue,
		collectedVotesChan:   make(chan []wire.Event, 1),
		ctx:                  ctx,
		regenerationChannel:  make(chan bool, 1),
		repropagationChannel: make(chan *bytes.Buffer, 100),
		lock:                 sync.Mutex{},
	}

	go wire.NewEventSubscriber(eventBus, collector, reductionTopic).Accept()
	go collector.onTimeout()
	return collector
}

func (c *collector) onTimeout() {
	for {
		<-c.ctx.timer.timeoutChan
		c.Clear()
	}
}

func (c *collector) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.StepEventCollector.Clear()
}

func (c *collector) Collect(buffer *bytes.Buffer) error {
	fmt.Println("Received a message")
	ev := c.ctx.handler.NewEvent()
	if err := c.ctx.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	if err := c.ctx.handler.Verify(ev); err != nil {
		return err
	}
	fmt.Println("message verified successfully")

	header := &consensus.EventHeader{}
	c.ctx.handler.ExtractHeader(ev, header)
	fmt.Println("current step is ", c.ctx.state.Step(), "message received for step", header.Step)
	// TODO: review re-propagation logic
	c.lock.Lock()
	if c.isRelevant(header.Round, header.Step) &&
		!c.Contains(ev, string(header.Step)) {
		// TODO: review
		c.repropagate(ev)
		c.process(ev)
		c.lock.Unlock()
		return nil
	}

	c.lock.Unlock()
	if c.isEarly(header.Round, header.Step) {
		fmt.Println("message is early")
		c.queue.PutEvent(header.Round, header.Step, ev)
	}

	return nil
}

func (c *collector) process(ev wire.Event) {
	b := new(bytes.Buffer)
	if err := c.ctx.handler.EmbedVoteHash(ev, b); err == nil {
		hash := hex.EncodeToString(b.Bytes())
		count := c.Store(ev, hash)
		if count > c.ctx.committee.Quorum() {
			votes := c.StepEventCollector[hash]
			c.collectedVotesChan <- votes
			c.Clear()
		}
	}
}

// TODO: review
func (c *collector) repropagate(ev wire.Event) {
	buf := new(bytes.Buffer)
	_ = c.ctx.handler.Marshal(buf, ev)
	c.repropagationChannel <- buf
}

func (c *collector) flushQueue() {
	queuedEvents := c.queue.GetEvents(c.ctx.state.Round(), c.ctx.state.Step())
	for _, event := range queuedEvents {
		c.process(event)
	}
}

func (c *collector) updateRound(round uint64) {
	c.ctx.state.Update(round)

	c.queue.Clear(c.ctx.state.Round())
	c.Clear()
	if c.reducer != nil {
		c.reducer.end()
		c.reducer = nil
	}
}

func (c *collector) isRelevant(round uint64, step uint8) bool {
	return c.ctx.state.Cmp(round, step) == 0 && c.reducer != nil
}

func (c *collector) isEarly(round uint64, step uint8) bool {
	return c.ctx.state.Cmp(round, step) > 0
}

func (c *collector) startReduction() {
	// TODO: review
	c.lock.Lock()
	c.reducer = newCoordinator(c.collectedVotesChan, c.ctx, c.regenerationChannel)

	go c.reducer.begin()
	c.lock.Unlock()
	c.flushQueue()
}

// newBroker will return a reduction broker.
// TODO: review regeneration
func newBroker(eventBus *wire.EventBus, handler handler, selectionChannel chan []byte,
	committee committee.Committee, reductionTopic topics.Topic, outgoingReductionTopic,
	outgoingAgreementTopic, generationTopic string, timeout time.Duration) *broker {

	ctx := newCtx(handler, committee, timeout)
	collector := newCollector(eventBus, string(reductionTopic), ctx)

	roundChannel := consensus.InitRoundUpdate(eventBus)
	return &broker{
		eventBus:               eventBus,
		roundUpdateChan:        roundChannel,
		unMarshaller:           newUnMarshaller(msg.VerifyEd25519Signature),
		ctx:                    ctx,
		collector:              collector,
		outgoingReductionTopic: outgoingReductionTopic,
		outgoingAgreementTopic: outgoingAgreementTopic,

		// TODO: review
		generationTopic: generationTopic,
		reductionTopic:  reductionTopic,
		selectionChan:   selectionChannel,
	}
}

// Listen for incoming messages.
func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundUpdateChan:
			b.collector.updateRound(round)
		case hash := <-b.selectionChan:
			b.forwardSelection(hash)
		case reductionVote := <-b.ctx.reductionVoteChan:
			b.eventBus.Publish(b.outgoingReductionTopic, reductionVote)
		case agreementVote := <-b.ctx.agreementVoteChan:
			b.eventBus.Publish(b.outgoingAgreementTopic, agreementVote)
		case <-b.collector.regenerationChannel:
			// TODO: remove
			fmt.Println("reduction finished")
			// TODO: review
			b.eventBus.Publish(b.generationTopic, nil)
		case ev := <-b.collector.repropagationChannel:
			// TODO: review
			message, _ := wire.AddTopic(ev, b.reductionTopic)
			b.eventBus.Publish(string(topics.Gossip), message)
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
	go b.collector.startReduction()
}
