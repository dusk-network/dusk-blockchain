package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
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
		Marshal(*bytes.Buffer, wire.Event) error
		Stage(wire.Event) (uint64, uint8)
		Hash(wire.Event) []byte
	}

	collector struct {
		collectedVotesChannel  chan []wire.Event
		reductionResultChannel chan []byte
		reductionVoteChannel   chan *bytes.Buffer
		agreementVoteChannel   chan *bytes.Buffer
		committee              committee.Committee
		timeOut                time.Duration
		handler                eventHandler

		reductionEventCollector
		currentRound uint64
		currentStep  uint8
		queue        *consensus.EventQueue
		voteStore    []wire.Event
		stopChannel  chan bool
		reducing     bool
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
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
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
		handler:                handler,
		queue:                  &queue,
		committee:              committee,
		timeOut:                timeOut,
		collectedVotesChannel:  make(chan []wire.Event, 1),
		reductionResultChannel: make(chan []byte, 1),
		reductionVoteChannel:   make(chan *bytes.Buffer, 1),
		agreementVoteChannel:   make(chan *bytes.Buffer, 1),
	}

	wire.NewEventSubscriber(eventBus, collector,
		string(topics.BlockReduction)).Accept()
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
	if c.isRelevant(round, step) {
		c.process(ev)
		return nil
	}

	if c.isEarly(round, step) {
		c.queue.PutEvent(round, step, ev)
	}

	return nil
}

func (c *collector) process(ev wire.Event) {
	hash := hex.EncodeToString(c.handler.Hash(ev))
	count := c.Store(ev, hash)
	if count > c.committee.Quorum() {
		votes := c.reductionEventCollector[hash]
		c.collectedVotesChannel <- votes
		c.Clear()
	}
}

func (c *collector) updateRound(round uint64) {
	c.queue.Clear(c.currentRound)
	c.stopReduction()
	c.stopReduction()
	c.currentRound = round
	c.currentStep = 1
}

func (c collector) isRelevant(round uint64, step uint8) bool {
	return c.currentRound == round && c.currentStep == step && c.reducing
}

func (c collector) isEarly(round uint64, step uint8) bool {
	return c.currentRound <= round || c.currentStep <= step
}

func (c *collector) startReduction() {
	// flush queue
	queuedEvents := c.queue.GetEvents(c.currentRound, c.currentStep)
	for _, event := range queuedEvents {
		c.process(event)
	}

	// start a timer
	timer := time.NewTimer(c.timeOut)
	select {
	case <-timer.C:
		c.stopReduction()
		c.reductionResultChannel <- make([]byte, 32)
	case votes := <-c.collectedVotesChannel:
		c.voteStore = append(c.voteStore, votes...)
		timer.Stop()
	}
}

// reduction is ran in segments of two steps. this function will listen for and
// handle both results.
func (c *collector) listenReduction() {
	c.reducing = true

	// after receiving the first result, send it to the broker as a reduction vote
	hash1 := <-c.reductionResultChannel
	c.currentStep++
	reductionVote, _ := c.addRoundAndStep(hash1)
	c.reductionVoteChannel <- reductionVote

	// after the second result, we check the voteStore and the two hashes.
	// if we have a good result, we send an agreement vote
	hash2 := <-c.reductionResultChannel
	if c.reductionSuccessful(hash1, hash2) {
		buf := bytes.NewBuffer(hash2)
		c.marshalVoteSet(buf)
		agreementVote, _ := c.addRoundAndStep(buf.Bytes())
		c.agreementVoteChannel <- agreementVote
	}

	c.currentStep++
	c.reducing = false
}

func (c *collector) stopReduction() {
	if c.reducing {
		c.collectedVotesChannel <- nil
		c.Clear()
	}
}

func (c collector) reductionSuccessful(hash1, hash2 []byte) bool {
	bothNotNil := hash1 != nil && hash2 != nil
	identicalResults := bytes.Equal(hash1, hash2)
	voteSetCorrectLength := len(c.voteStore) >= c.committee.Quorum()*2

	return bothNotNil && identicalResults && voteSetCorrectLength
}

func (c collector) addRoundAndStep(data []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.WriteUint64(buffer, binary.LittleEndian, c.currentRound); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, c.currentStep); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(data); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (c collector) marshalVoteSet(r *bytes.Buffer) error {
	if err := encoding.WriteVarInt(r, uint64(len(c.voteStore))); err != nil {
		return err
	}

	for _, event := range c.voteStore {
		if err := c.handler.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
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
		eventBus:         eventBus,
		collector:        collector,
		selectionChannel: selectionChannel,
		roundChannel:     roundCollector.RoundChan,
	}
}

// Listen for incoming messages.
func (b *Broker) Listen() {
	for {
		select {
		case round := <-b.roundChannel:
			b.updateRound(round)
		case hash := <-b.selectionChannel:
			reductionVote, _ := b.addRoundAndStep(hash)
			b.eventBus.Publish(msg.OutgoingReductionTopic, reductionVote)
			go b.listenReduction()
			go b.startReduction()
		case reductionVote := <-b.reductionVoteChannel:
			b.eventBus.Publish(msg.OutgoingReductionTopic, reductionVote)
			go b.startReduction()
		case agreementVote := <-b.agreementVoteChannel:
			b.eventBus.Publish(msg.OutgoingAgreementTopic, agreementVote)
		}
	}
}
