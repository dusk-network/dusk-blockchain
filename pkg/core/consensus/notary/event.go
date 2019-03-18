package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// A committeeEvent is an embeddable struct that groups common operations on notaries
type committeeEvent struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	Round         uint64
	Step          uint8
	BlockHash     []byte
}

// Equal as specified in the Event interface
func (a *committeeEvent) Equal(e wire.Event) bool {
	other, ok := e.(*committeeEvent)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

type committeeEventUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

func newCommitteeEventUnmarshaller(validate func(*bytes.Buffer) error) *committeeEventUnmarshaller {
	return &committeeEventUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *committeeEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	committeeEvent := ev.(*committeeEvent)

	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return err
	}
	committeeEvent.VoteSet = voteSet

	if err := encoding.ReadBLS(r, &committeeEvent.SignedVoteSet); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &committeeEvent.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &committeeEvent.BlockHash); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &committeeEvent.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &committeeEvent.Step); err != nil {
		return err
	}

	return nil
}

// CommitteeCollector is a helper that groups common operations performed on Events related to a committee
type CommitteeCollector struct {
	wire.StepEventCollector
	committee    user.Committee
	currentRound uint64
}

// ShouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (cc *CommitteeCollector) ShouldBeSkipped(m *committeeEvent) bool {
	isDupe := cc.Contains(m, m.Step)
	isPleb := !cc.committee.IsMember(m.PubKeyBLS)
	//TODO: the round element needs to be reassessed
	err := cc.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
	failedVerification := err != nil
	return isDupe || isPleb || failedVerification
}
