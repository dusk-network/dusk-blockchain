package agreement

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type CommitteeEvent struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	Round         uint64
	Step          uint8
	BlockHash     []byte
	validate      func(*bytes.Buffer) error
}

func (a *CommitteeEvent) Equal(e Event) bool {
	other, ok := e.(*CommitteeEvent)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

func newCommiteeEvent(validate func(*bytes.Buffer) error) *CommitteeEvent {
	return &CommitteeEvent{
		validate: validate,
	}
}

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
	err := cc.Verify(m)
	failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant || failedVerification
}

func (cc *CommitteeCollector) Verify(m *CommitteeEvent) error {
	return cc.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
}

type BlockCollector struct {
	*CommitteeCollector
	blockChan chan<- *CommitteeEvent
}

func NewBlockCollector(committee user.Committee, blockChan chan *CommitteeEvent, validateFunc func(*bytes.Buffer) error) *BlockCollector {

	cc := &CommitteeCollector{
		committee:    committee,
		validateFunc: validateFunc,
	}

	return &BlockCollector{
		CommitteeCollector: cc,
		blockChan:          blockChan,
	}
}

func (c *BlockCollector) UpdateRound(round uint64) {
	c.currentRound = round
}

func (c *BlockCollector) Collect(buffer *bytes.Buffer) error {
	ev := newCommiteeEvent(c.validateFunc)
	if err := c.GetCommitteeEvent(buffer, ev); err != nil {
		return err
	}

	if !c.ShouldBeSkipped(ev) {
		c.Process(ev)
	}
	return nil
}

func (c *BlockCollector) Process(event *CommitteeEvent) {
	nrAgreements := c.Store(event, event.Step)
	// did we reach the quorum?
	if nrAgreements >= c.committee.Quorum() {
		// notify the Notary
		go func() { c.blockChan <- event }()
		c.Clear()
	}
}

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	eventBus        *wire.EventBus
	blockSubscriber *EventSubscriber
	roundSubscriber *EventSubscriber
	blockChan       <-chan *CommitteeEvent
	roundChan       <-chan uint64
	blockCollector  *BlockCollector
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus,
	validateFunc func(*bytes.Buffer) error,
	committee *user.CommitteeStore) *BlockNotary {

	blockChan := make(chan *CommitteeEvent, 1)
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
		case message := <-b.blockChan:
			buffer := bytes.NewBuffer(message.BlockHash)
			b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
		case round := <-b.roundChan:
			b.blockCollector.UpdateRound(round)
		}
	}
}

type RoundCollector struct {
	roundChan chan uint64
}

func (r *RoundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}
