package reduction

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// Event is a basic reduction event.
	Event struct {
		*consensus.EventHeader
		VotedHash  []byte
		SignedHash []byte
	}

	eventHandler interface {
		wire.EventVerifier
		NewEvent() wire.Event
		Unmarshal(*bytes.Buffer, wire.Event) error
		Stage(wire.Event) (uint64, uint8)
		Hash(wire.Event) string
	}

	collector struct {
		collectedVotesChannel chan []wire.Event
		committee             committee.Committee
		timeOut               time.Duration
		handler               eventHandler

		reductionEventCollector
		currentRound uint64
		currentStep  uint8
		queue        *consensus.EventQueue
		voteStore    []wire.Event
		stopChannel  chan bool
	}

	// Broker is the message broker for the reduction process.
	Broker struct {
		eventBus *wire.EventBus
		*collector

		// channels linked to subscribers
		roundChannel       <-chan uint64
		phaseUpdateChannel <-chan []byte
		selectionChannel   <-chan []byte
	}

	selectionCollector struct {
		selectionChannel chan<- []byte
	}
)

// Equal as specified in the Event interface
func (e *Event) Equal(ev wire.Event) bool {
	other, ok := ev.(*Event)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) && (e.Round == other.Round) && (e.Step == other.Step)
}

// Collect implements the EventCollector interface.
// Will simply send the received buffer as a slice of bytes.
func (s selectionCollector) Collect(buffer *bytes.Buffer) error {
	s.selectionChannel <- buffer.Bytes()
	return nil
}

func newCollector(eventBus *wire.EventBus, committee committee.Committee,
	validateFunc func(*bytes.Buffer) error, handler eventHandler,
	timeOut time.Duration) *collector {

	queue := consensus.NewEventQueue()
	collector := &collector{
		handler:   handler,
		queue:     &queue,
		committee: committee,
		timeOut:   timeOut,
	}

	wire.NewEventSubscriber(eventBus, collector, string(topics.BlockReduction)).Accept()
	return collector
}

// Collect implements the EventCollector interface.
// Unmarshal a reduction message, verify the signature and then
// pass it down for processing.
func (c *collector) Collect(buffer *bytes.Buffer) error {
	ev := c.handler.NewEvent()
	if err := c.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	if err := c.handler.Verify(ev); err != nil {
		return err
	}

	round, step := c.handler.Stage(ev)
	if c.isEarly(round, step) {
		c.queue.PutEvent(round, step, ev)
		return nil
	}

	if c.isRelevant(round, step) {
		c.process(ev)
	}

	return nil
}

func (c *collector) process(ev wire.Event) {
	hash := c.handler.Hash(ev)
	count := c.Store(ev, hash)
	if count > c.committee.Quorum() {
		votes := c.reductionEventCollector[hash]
		c.collectedVotesChannel <- votes
		c.Clear()
	}
}

func (c *collector) updateRound(round uint64) {
	c.queue.Clear(c.currentRound)
	c.currentRound = round
	c.currentStep = 1
	c.Clear()
}

func (c collector) isRelevant(round uint64, step uint8) bool {
	return c.currentRound == round && c.currentStep == step
}

func (c collector) isEarly(round uint64, step uint8) bool {
	return c.currentRound <= round || c.currentStep <= step
}

// NewBroker will return a reduction broker.
func NewBroker(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	handler eventHandler, committee committee.Committee,
	timeOut time.Duration) *Broker {

	collector := newCollector(eventBus, committee, validateFunc, handler, timeOut)

	selectionChannel := make(chan []byte, 1)
	selectionCollector := selectionCollector{selectionChannel}
	wire.NewEventSubscriber(eventBus, selectionCollector,
		msg.SelectionResultTopic).Accept()

	roundCollector := consensus.InitRoundCollector(eventBus)

	return &Broker{
		eventBus:     eventBus,
		collector:    collector,
		roundChannel: roundCollector.RoundChan,
	}
}

// Listen for incoming messages.
func (b *Broker) Listen() {
	for {
		select {
		case round := <-b.roundChannel:
			b.updateRound(round)
		case <-b.selectionChannel:
			// start reduction
		}
	}
}

func (b Broker) vote(data []byte, topic string) error {
	voteData, err := b.addVoteInfo(data)
	if err != nil {
		return err
	}

	b.eventBus.Publish(topic, voteData)
	return nil
}

func (b Broker) addVoteInfo(data []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.WriteUint64(buffer, binary.LittleEndian, b.currentRound); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, b.currentStep); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(data); err != nil {
		return nil, err
	}

	return buffer, nil
}
