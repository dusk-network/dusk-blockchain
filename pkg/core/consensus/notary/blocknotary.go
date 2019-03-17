package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// BlockEvent expresses a vote on a block hash. It is a real type alias of committeeEvent
type BlockEvent = committeeEvent

// BlockEventUnmarshaller is the unmarshaller of BlockEvents. It is a real type alias of committeeEventUnmarshaller
type BlockEventUnmarshaller = committeeEventUnmarshaller

// BlockCollector collects CommitteeEvent. When a Quorum is reached, it propagates the new Block Hash to the proper channel
type BlockCollector struct {
	*CommitteeCollector
	blockChan    chan<- []byte
	Unmarshaller EventUnmarshaller
}

// NewBlockCollector is injected with the committee, a channel where to publish the new Block Hash and the validator function for shallow checking of the marshalled form of the CommitteeEvent messages.
func NewBlockCollector(committee user.Committee, blockChan chan []byte, validateFunc func(*bytes.Buffer) error) *BlockCollector {

	cc := &CommitteeCollector{
		StepEventCollector: make(map[uint8][]Event),
		committee:          committee,
	}

	return &BlockCollector{
		CommitteeCollector: cc,
		blockChan:          blockChan,
		Unmarshaller:       newCommitteeEventUnmarshaller(validateFunc),
	}
}

// UpdateRound is used to change the current round
func (c *BlockCollector) UpdateRound(round uint64) {
	c.currentRound = round
}

// Collect as specifiec in the EventCollector interface. It dispatches the unmarshalled CommitteeEvent to Process method
func (c *BlockCollector) Collect(buffer *bytes.Buffer) error {
	ev := &BlockEvent{}
	if err := c.Unmarshaller.Unmarshal(buffer, ev); err != nil {
		return err
	}

	isIrrelevant := c.currentRound != 0 && c.currentRound != ev.Round
	if c.ShouldBeSkipped(ev) || isIrrelevant {
		return nil
	}

	c.Process(ev)
	return nil
}

// Process checks if the quorum is reached and if it isn't simply stores the Event in the proper step
func (c *BlockCollector) Process(event *BlockEvent) {
	nrAgreements := c.Store(event, event.Step)
	// did we reach the quorum?
	if nrAgreements >= c.committee.Quorum() {
		// notify the Notary
		go func() { c.blockChan <- event.BlockHash }()
		c.Clear()
	}
}

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	eventBus        *wire.EventBus
	blockSubscriber *EventSubscriber
	roundSubscriber *EventSubscriber
	blockChan       <-chan []byte
	roundChan       <-chan uint64
	blockCollector  *BlockCollector
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus,
	validateFunc func(*bytes.Buffer) error,
	committee user.Committee) *BlockNotary {

	blockChan := make(chan []byte, 1)
	roundChan := make(chan uint64, 1)

	blockCollector := NewBlockCollector(committee, blockChan, validateFunc)
	blockSubscriber := NewEventSubscriber(eventBus,
		blockCollector,
		string(msg.BlockAgreementTopic))

	roundCollector := &RoundCollector{roundChan}
	roundSubscriber := NewEventSubscriber(eventBus, roundCollector, string(msg.RoundUpdateTopic))

	return &BlockNotary{
		eventBus:        eventBus,
		blockSubscriber: blockSubscriber,
		roundSubscriber: roundSubscriber,
		blockChan:       blockChan,
		roundChan:       roundChan,
		blockCollector:  blockCollector,
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// BlockNotary gets a currentRound somehow
func (b *BlockNotary) Listen() {
	go b.blockSubscriber.Accept()
	go b.roundSubscriber.Accept()
	for {
		select {
		case blockHash := <-b.blockChan:
			buffer := bytes.NewBuffer(blockHash)
			b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
		case round := <-b.roundChan:
			b.blockCollector.UpdateRound(round)
		}
	}
}

// RoundCollector is a simple wrapper over a channel to get round notifications
type RoundCollector struct {
	roundChan chan uint64
}

// Collect as specified in the EventCollector interface. In this case Collect simply performs unmarshalling of the round event
func (r *RoundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}
