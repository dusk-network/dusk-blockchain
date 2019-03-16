package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// CommitteeEvent expresses a vote on a block hash
type CommitteeEvent struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	Round         uint64
	Step          uint8
	BlockHash     []byte
	validate      func(*bytes.Buffer) error
}

// Equal as specified in the Event interface
func (a *CommitteeEvent) Equal(e Event) bool {
	other, ok := e.(*CommitteeEvent)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

// Create a CommitteeEvent
func newCommiteeEvent(validate func(*bytes.Buffer) error) *CommitteeEvent {
	return &CommitteeEvent{
		validate: validate,
	}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *CommitteeEvent) Unmarshal(r *bytes.Buffer) error {
	if err := a.validate(r); err != nil {
		return err
	}

	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return err
	}
	a.VoteSet = voteSet

	if err := encoding.ReadBLS(r, &a.SignedVoteSet); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &a.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &a.BlockHash); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &a.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &a.Step); err != nil {
		return err
	}

	return nil
}

// CommitteeCollector is a helper that groups common operations performed on Events related to a committee
type CommitteeCollector struct {
	StepEventCollector
	committee    user.Committee
	currentRound uint64
	validateFunc func(*bytes.Buffer) error
}

// ShouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (cc *CommitteeCollector) ShouldBeSkipped(m *CommitteeEvent) bool {
	isDupe := cc.Contains(m, m.Step)
	isPleb := !cc.committee.IsMember(m.PubKeyBLS)
	//TODO: the round element needs to be reassessed
	isIrrelevant := cc.currentRound != 0 && cc.currentRound < m.Round
	err := cc.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
	failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant || failedVerification
}

// BlockCollector collects CommitteeEvent. When a Quorum is reached, it propagates the new Block Hash to the proper channel
type BlockCollector struct {
	*CommitteeCollector
	blockChan chan<- []byte
}

// NewBlockCollector is injected with the committee, a channel where to publish the new Block Hash and the validator function for shallow checking of the marshalled form of the CommitteeEvent messages
func NewBlockCollector(committee user.Committee, blockChan chan []byte, validateFunc func(*bytes.Buffer) error) *BlockCollector {

	cc := &CommitteeCollector{
		StepEventCollector: make(map[uint8][]Event),
		committee:          committee,
		validateFunc:       validateFunc,
	}

	return &BlockCollector{
		CommitteeCollector: cc,
		blockChan:          blockChan,
	}
}

// UpdateRound is used to change the current round
func (c *BlockCollector) UpdateRound(round uint64) {
	c.currentRound = round
}

// Collect as specifiec in the EventCollector interface. It dispatches the unmarshalled CommitteeEvent to Process method
func (c *BlockCollector) Collect(buffer *bytes.Buffer) error {
	ev := newCommiteeEvent(c.validateFunc)
	if err := ev.Unmarshal(buffer); err != nil {
		return err
	}

	if !c.ShouldBeSkipped(ev) {
		c.Process(ev)
	}
	return nil
}

// Process checks if the quorum is reached and if it isn't simply stores the Event in the proper step
func (c *BlockCollector) Process(event *CommitteeEvent) {
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
	committee *user.CommitteeStore) *BlockNotary {

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
