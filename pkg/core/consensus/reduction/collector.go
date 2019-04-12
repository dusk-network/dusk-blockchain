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
		queue     *consensus.EventQueue
		eventChan chan<- wire.Event
		ctx       *context
	}

	// Broker is the message broker for the reduction process.
	broker struct {
		collector   *collector
		accumulator *accumulator
		// utility context to group interfaces and channels to be passed around
		ctx *context

		// channels linked to subscribers
		roundUpdateChan <-chan uint64
		selectionChan   <-chan []byte

		// utility
		unMarshaller *unMarshaller
	}
)

func launchCollector(subscriber wire.EventSubscriber, topic string, ctx *context) (*collector, <-chan wire.Event) {
	eventChan := make(chan wire.Event)
	collector := newCollector(ctx, eventChan)
	go wire.NewTopicListener(subscriber, collector, topic).Accept()
	return collector, eventChan
}

func newCollector(ctx *context, eventChan chan<- wire.Event) *collector {
	collector := &collector{
		queue:     consensus.NewEventQueue(),
		ctx:       ctx,
		eventChan: eventChan,
	}

	return collector
}

func (c *collector) Collect(buffer *bytes.Buffer) error {
	ev := c.ctx.handler.NewEvent()
	if err := c.ctx.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	header := &consensus.EventHeader{}
	c.ctx.handler.ExtractHeader(ev, header)
	if c.isRelevant(header.Round, header.Step) {
		c.eventChan <- ev
		return nil
	}

	if c.isEarly(header.Round, header.Step) {
		c.queue.PutEvent(header.Round, header.Step, ev)
	}

	return nil
}

func (c *collector) flushQueue() {
	queuedEvents := c.queue.GetEvents(c.ctx.state.Round(), c.ctx.state.Step())
	for _, event := range queuedEvents {
		c.eventChan <- event
	}
}

func (c *collector) updateRound(round uint64) {
	c.queue.Clear(round - 1)
}

func (c *collector) isRelevant(round uint64, step uint8) bool {
	relevant := c.ctx.state.Cmp(round, step) == 0
	if !relevant {
		log.WithFields(log.Fields{
			"process": "reducer",
			"round":   round,
			"step":    step,
			"state":   c.ctx.state.String(),
		}).Debugln("isRelevant mismatch")
	}
	return relevant
}

func (c *collector) isEarly(round uint64, step uint8) bool {
	return c.ctx.state.Cmp(round, step) <= 0
}

// newBroker will return a reduction broker.
func newBroker(eventBroker wire.EventBroker, handler handler, selectionChannel chan []byte, committee committee.Committee, timeout time.Duration) *broker {
	ctx := newCtx(handler, committee, timeout)
	collector, eventChan := launchCollector(eventBroker, string(topics.BlockReduction), ctx)
	accumulator := launchAccumulator(ctx, eventBroker, eventChan)

	roundChannel := consensus.InitRoundUpdate(eventBroker)
	return &broker{
		roundUpdateChan: roundChannel,
		unMarshaller:    newUnMarshaller(msg.VerifyEd25519Signature),
		ctx:             ctx,
		collector:       collector,
		accumulator:     accumulator,
		selectionChan:   selectionChannel,
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
				"round":   b.collector.ctx.state.Round(),
				"hash":    hex.EncodeToString(hash),
			}).Debug("Got selection message")
			b.collector.flushQueue()
			b.accumulator.startReduction(hash)
		}
	}
}
