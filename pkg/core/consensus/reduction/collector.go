package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	handler interface {
		consensus.EventHandler
		// TODO: not sure we need this
		EmbedVoteHash(wire.Event, *bytes.Buffer) error
		MarshalHeader(r *bytes.Buffer, state *consensusState) error
		MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error
	}

	consensusState struct {
		Round uint64
		Step  uint8
	}

	unMarshaller struct {
		*consensus.EventHeaderUnmarshaller
		*consensus.EventHeaderMarshaller
	}

	collector struct {
		consensus.StepEventCollector
		collectedVotesChan  chan []wire.Event
		reductionResultChan chan *bytes.Buffer
		reductionVoteChan   chan *bytes.Buffer
		agreementVoteChan   chan *bytes.Buffer

		committee   committee.Committee
		timeout     time.Duration
		handler     handler
		state       *consensusState
		queue       *consensus.EventQueue
		coordinator *coordinator
	}

	// Broker is the message broker for the reduction process.
	Broker struct {
		eventBus  *wire.EventBus
		collector *collector

		// channels linked to subscribers
		roundUpdateChan <-chan uint64
		phaseUpdateChan <-chan []byte
		selectionChan   <-chan *bytes.Buffer

		// utility
		unMarshaller *unMarshaller
	}

	selectionCollector struct {
		selectionChan chan<- *bytes.Buffer
	}
)

func newUnMarshaller() *unMarshaller {
	return &unMarshaller{
		EventHeaderUnmarshaller: consensus.NewEventHeaderUnmarshaller(),
		EventHeaderMarshaller:   &consensus.EventHeaderMarshaller{},
	}
}

func (a *unMarshaller) MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := a.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

func (a *unMarshaller) MarshalHeader(r *bytes.Buffer, state *consensusState) error {
	buffer := new(bytes.Buffer)
	// Decoding Round
	if err := encoding.ReadUint64(r, binary.LittleEndian, &state.Round); err != nil {
		return err
	}

	// Decoding Step
	if err := encoding.ReadUint8(r, &state.Step); err != nil {
		return err
	}

	if _, err := buffer.Write(r.Bytes()); err != nil {
		return err
	}
	return nil
}

// Collect implements the EventCollector interface.
// Will simply send the received buffer as a slice of bytes.
func (s selectionCollector) Collect(buffer *bytes.Buffer) error {
	s.selectionChan <- buffer
	return nil
}

func newCollector(eventBus *wire.EventBus, committee committee.Committee, state *consensusState, handler handler, reductionTopic string, timeout time.Duration) *collector {

	queue := consensus.NewEventQueue()
	collector := &collector{
		handler:             handler,
		queue:               &queue,
		committee:           committee,
		state:               state,
		timeout:             timeout,
		collectedVotesChan:  make(chan []wire.Event, 1),
		reductionResultChan: make(chan *bytes.Buffer, 1),
		reductionVoteChan:   make(chan *bytes.Buffer, 1),
		agreementVoteChan:   make(chan *bytes.Buffer, 1),
	}

	wire.NewEventSubscriber(eventBus, collector, reductionTopic).Accept()
	return collector
}

func (c *collector) Collect(buffer *bytes.Buffer) error {
	ev := c.handler.NewEvent()
	if err := c.handler.Unmarshal(buffer, ev); err != nil {
		return err
	}

	if err := c.handler.Verify(ev); err != nil {
		return err
	}

	header := &consensus.EventHeader{}
	c.handler.ExtractHeader(ev, header)
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
	c.handler.EmbedVoteHash(ev, bytes.NewBuffer(b))
	hash := hex.EncodeToString(b)
	count := c.Store(ev, hash)
	if count > c.committee.Quorum() {
		votes := c.StepEventCollector[hash]
		c.collectedVotesChan <- votes
		c.Clear()
	}
}

func (c collector) flushQueue() {
	queuedEvents := c.queue.GetEvents(c.state.Round, c.state.Step)
	for _, event := range queuedEvents {
		c.process(event)
	}
}

func (c *collector) updateRound(round uint64) {
	c.state.Round = round
	c.state.Step = 1

	c.queue.Clear(c.state.Round)
	c.Clear()
	if c.coordinator != nil {
		c.coordinator.end()
		c.coordinator = nil
	}
}

func (c collector) isRelevant(round uint64, step uint8) bool {
	return c.state.Round == round && c.state.Step == step && c.coordinator != nil
}

func (c collector) isEarly(round uint64, step uint8) bool {
	return c.state.Round < round || c.state.Round == round && c.state.Step < step
}

func (c *collector) startReduction() {
	c.coordinator = newCoordinator(c.state, c.collectedVotesChan, c.reductionVoteChan, c.agreementVoteChan, c.handler, c.committee)

	go c.flushQueue()
	go c.coordinator.begin(c.timeout)
}

// NewBroker will return a reduction broker.
func NewBroker(eventBus *wire.EventBus,
	handler handler, committee committee.Committee, selectionTopic,
	reductionTopic string, timeout time.Duration) *Broker {

	state := &consensusState{}

	collector := newCollector(eventBus, committee, state, handler, reductionTopic, timeout)

	selectionChan := make(chan *bytes.Buffer, 1)
	selectionCollector := selectionCollector{
		selectionChan: selectionChan,
	}
	go wire.NewEventSubscriber(eventBus, selectionCollector,
		selectionTopic).Accept()

	roundChannel := consensus.InitRoundUpdate(eventBus)

	return &Broker{
		eventBus:        eventBus,
		collector:       collector,
		selectionChan:   selectionChan,
		roundUpdateChan: roundChannel,
		unMarshaller:    newUnMarshaller(),
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
			b.unMarshaller.MarshalHeader(buf, b.collector.state)
			b.eventBus.Publish(msg.OutgoingReductionTopic, buf)
			go b.collector.startReduction()
		case reductionVote := <-b.collector.reductionVoteChan:
			b.eventBus.Publish(msg.OutgoingReductionTopic, reductionVote)
			// the second reduction step is triggered by a reductionVote result
			go b.collector.startReduction()
		case agreementVote := <-b.collector.agreementVoteChan:
			b.eventBus.Publish(msg.OutgoingAgreementTopic, agreementVote)
		}
	}
}
