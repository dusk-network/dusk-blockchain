package reduction

import (
	"bytes"
	"encoding/hex"
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
		queue        *consensus.EventQueue
		eventChan    chan<- wire.Event
		handler      consensus.EventHandler
		currentState consensus.State
		bouncer      *bouncer
	}

	// Broker is the message broker for the reduction process.
	broker struct {
		collector   *collector
		accumulator *accumulator
		// utility context to group interfaces and channels to be passed around
		ctx *context

		// channels linked to subscribers
		roundUpdateChan <-chan uint64
		stateChan       <-chan struct{}
		selectionChan   <-chan []byte

		// utility
		unMarshaller *unMarshaller
	}
)

func launchCollector(subscriber wire.EventSubscriber, topic string, handler consensus.EventHandler,
	state consensus.State, bouncer *bouncer) (*collector, <-chan wire.Event) {

	eventChan := make(chan wire.Event)
	collector := newCollector(handler, eventChan, state, bouncer)
	go wire.NewTopicListener(subscriber, collector, topic).Accept()
	return collector, eventChan
}

func newCollector(handler consensus.EventHandler, eventChan chan<- wire.Event,
	state consensus.State, bouncer *bouncer) *collector {
	collector := &collector{
		queue:        consensus.NewEventQueue(),
		handler:      handler,
		eventChan:    eventChan,
		currentState: state,
		bouncer:      bouncer,
	}

	return collector
}

func (c *collector) Collect(buffer *bytes.Buffer) error {
	ev := c.handler.NewEvent()
	if err := c.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	c.bouncer.repropagate(ev)
	header := &consensus.EventHeader{}
	c.handler.ExtractHeader(ev, header)
	if c.isEarly(header.Round, header.Step) {
		c.queue.PutEvent(header.Round, header.Step, ev)
		return nil
	}

	c.eventChan <- ev
	return nil
}

func (c *collector) flushQueue() {
	queuedEvents := c.queue.GetEvents(c.currentState.Round(), c.currentState.Step())
	for _, event := range queuedEvents {
		c.eventChan <- event
	}
}

func (c *collector) updateRound(round uint64) {
	c.currentState.Update(round)
	c.queue.Clear(round - 1)
	log.WithFields(log.Fields{
		"process": "reduction",
		"state":   c.currentState.String(),
	}).Traceln("state updated")
}

func (c *collector) isEarly(round uint64, step uint8) bool {
	early := c.currentState.Cmp(round, step) < 0
	if early {
		log.WithFields(log.Fields{
			"process": "reducer",
			"round":   round,
			"step":    step,
			"state":   c.currentState.String(),
		}).Debugln("queueing message")
	}
	return early
}

// newBroker will return a reduction broker.
func newBroker(eventBroker wire.EventBroker, handler handler, selectionChannel chan []byte, committee committee.Committee, timeout time.Duration) *broker {
	ctx := newCtx(handler, committee, timeout)
	bouncer := newBouncer(eventBroker, handler)
	collector, eventChan := launchCollector(eventBroker, string(topics.BlockReduction), handler, ctx.state, bouncer)
	accumulator := launchAccumulator(ctx, eventBroker, eventChan)

	roundChannel := consensus.InitRoundUpdate(eventBroker)
	stateSub := ctx.state.SubscribeState()
	return &broker{
		roundUpdateChan: roundChannel,
		unMarshaller:    newUnMarshaller(msg.VerifyEd25519Signature),
		ctx:             ctx,
		collector:       collector,
		accumulator:     accumulator,
		selectionChan:   selectionChannel,
		stateChan:       stateSub.StateChan,
	}
}

// Listen for incoming messages.
func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundUpdateChan:
			b.collector.updateRound(round)
			b.accumulator.updateRound(round)
		case hash := <-b.selectionChan:
			log.WithFields(log.Fields{
				"process": "reduction",
				"round":   b.ctx.state.Round(),
				"hash":    hex.EncodeToString(hash),
			}).Debug("Got selection message")
			b.collector.flushQueue()
			b.accumulator.startReduction(hash)
		case <-b.stateChan:
			b.collector.flushQueue()
		}
	}
}
