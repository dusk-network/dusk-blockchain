package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	collector struct {
		consensus.StepEventCollector
		collectedVotesChan chan []wire.Event
		queue              *consensus.EventQueue
		coordinator        *coordinator
		ctx                *context
	}

	// Broker is the message broker for the reduction process.
	Broker struct {
		eventBus  *wire.EventBus
		collector *collector
		// utility context to group interfaces and channels to be passed around
		ctx *context

		// channels linked to subscribers
		roundUpdateChan <-chan uint64
		selectionChan   <-chan *bytes.Buffer

		// utility
		unMarshaller *unMarshaller
	}

	selectionCollector struct {
		selectionChan chan<- *bytes.Buffer
	}
)

// Collect implements the EventCollector interface.
// Will simply send the received buffer as a slice of bytes.
func (s selectionCollector) Collect(buffer *bytes.Buffer) error {
	s.selectionChan <- buffer
	return nil
}

func newCollector(eventBus *wire.EventBus, reductionTopic string, ctx *context) *collector {

	queue := consensus.NewEventQueue()
	collector := &collector{
		queue:              &queue,
		collectedVotesChan: make(chan []wire.Event, 1),
		ctx:                ctx,
	}

	wire.NewEventSubscriber(eventBus, collector, reductionTopic).Accept()
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
	b := make([]byte, 0, 32)
	// TODO: for the sigset reduction the hash is actually the blockhash and the voteHash. Check this
	if err := c.ctx.handler.EmbedVoteHash(ev, bytes.NewBuffer(b)); err == nil {
		hash := hex.EncodeToString(b)
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
	if c.coordinator != nil {
		c.coordinator.end()
		c.coordinator = nil
	}
}

func (c collector) isRelevant(round uint64, step uint8) bool {
	return c.ctx.state.Round == round && c.ctx.state.Step == step && c.coordinator != nil
}

func (c collector) isEarly(round uint64, step uint8) bool {
	return c.ctx.state.Round < round || c.ctx.state.Round == round && c.ctx.state.Step < step
}

func (c *collector) startReduction() {
	c.coordinator = newCoordinator(c.collectedVotesChan, c.ctx)

	go c.flushQueue()
	// TODO: what to do with errors?
	go c.coordinator.begin()
}

// NewBroker will return a reduction broker.
func NewBroker(eventBus *wire.EventBus,
	handler handler, committee committee.Committee, selectionTopic,
	reductionTopic string, timeout time.Duration) *Broker {

	ctx := newCtx(handler, committee, timeout)
	collector := newCollector(eventBus, reductionTopic, ctx)

	selectionChan := make(chan *bytes.Buffer, 1)
	selectionCollector := selectionCollector{
		selectionChan: selectionChan,
	}
	go wire.NewEventSubscriber(eventBus, selectionCollector,
		selectionTopic).Accept()

	roundChannel := consensus.InitRoundUpdate(eventBus)

	return &Broker{
		eventBus:        eventBus,
		selectionChan:   selectionChan,
		roundUpdateChan: roundChannel,
		unMarshaller:    newUnMarshaller(),
		ctx:             ctx,
		collector:       collector,
	}
}

// Listen for incoming messages.
func (b *Broker) Listen() {
	for {
		select {
		case round := <-b.roundUpdateChan:
			b.collector.updateRound(round)
		case buf := <-b.selectionChan:
			// TODO: make sure we get only one hash instead of a full Event. Maybe consider using Marshal
			// the first reduction step is triggered by a sigSetSelection message
			_ = b.unMarshaller.MarshalHeader(buf, b.ctx.state)
			b.eventBus.Publish(msg.OutgoingReductionTopic, buf)
			go b.collector.startReduction()
		case reductionVote := <-b.ctx.reductionVoteChan:
			b.eventBus.Publish(msg.OutgoingReductionTopic, reductionVote)
			// the second reduction step is triggered by a reductionVote result
			go b.collector.startReduction()
		case agreementVote := <-b.ctx.agreementVoteChan:
			b.eventBus.Publish(msg.OutgoingAgreementTopic, agreementVote)
		}
	}
}
