package reduction

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	collector struct {
		consensus.StepEventCollector
		collectedVotesChan chan []wire.Event
		queue              *consensus.EventQueue
		reducer            *reducer
		ctx                *context

		// TODO: review this after demo. used to restart phase after reduction
		regenerationChannel chan bool
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
		regenerationTopic string
	}
)

func newCollector(eventBus *wire.EventBus, reductionTopic string, ctx *context) *collector {

	queue := consensus.NewEventQueue()
	collector := &collector{
		StepEventCollector:  consensus.StepEventCollector{},
		queue:               &queue,
		collectedVotesChan:  make(chan []wire.Event, 1),
		ctx:                 ctx,
		regenerationChannel: make(chan bool, 1),
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

func (c *collector) Collect(buffer *bytes.Buffer) error {
	ev := c.ctx.handler.NewEvent()
	if err := c.ctx.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	if err := c.ctx.handler.Verify(ev); err != nil {
		return err
	}

	header := &consensus.EventHeader{}
	c.ctx.handler.ExtractHeader(ev, header)
	if c.isRelevant(header.Round, header.Step) {
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
		count := c.Store(ev, hash)
		if count > c.ctx.committee.Quorum() {
			votes := c.StepEventCollector[hash]
			c.collectedVotesChan <- votes
			c.Clear()
		}
	}
}

func (c collector) flushQueue() {
	queuedEvents := c.queue.GetEvents(c.ctx.state.Round, c.ctx.state.Step)
	for _, event := range queuedEvents {
		c.process(event)
	}
}

func (c *collector) updateRound(round uint64) {
	c.ctx.state.Round = round
	c.ctx.state.Step = 1

	c.queue.Clear(c.ctx.state.Round)
	c.Clear()
	if c.reducer != nil {
		c.reducer.end()
		c.reducer = nil
	}
}

func (c collector) isRelevant(round uint64, step uint8) bool {
	return c.ctx.state.Round == round && c.ctx.state.Step == step && c.reducer != nil
}

func (c collector) isEarly(round uint64, step uint8) bool {
	return c.ctx.state.Round < round || c.ctx.state.Round == round && c.ctx.state.Step < step
}

func (c *collector) startReduction() {
	// TODO: review
	c.reducer = newCoordinator(c.collectedVotesChan, c.ctx, c.regenerationChannel)

	go c.flushQueue()
	go c.reducer.begin()
}

// newBroker will return a reduction broker.
// TODO: review regeneration
func newBroker(eventBus *wire.EventBus, handler handler, selectionChannel chan []byte,
	committee committee.Committee, reductionTopic, outgoingReductionTopic,
	outgoingAgreementTopic, regenerationTopic string, timeout time.Duration) *broker {

	ctx := newCtx(handler, committee, timeout)
	collector := newCollector(eventBus, reductionTopic, ctx)

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
		regenerationTopic: regenerationTopic,
		selectionChan:     selectionChannel,
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
			b.eventBus.Publish(b.regenerationTopic, nil)
		}
	}
}

func (b *broker) forwardSelection(hash []byte) {
	buf := bytes.NewBuffer(hash)
	if err := b.ctx.handler.MarshalHeader(buf, b.ctx.state); err != nil {
		panic(err)
	}
	b.eventBus.Publish(b.outgoingReductionTopic, buf)
	go b.collector.startReduction()
}
