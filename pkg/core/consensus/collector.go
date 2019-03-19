package consensus

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// EventHeader is an embeddable struct representing the consensus event header fields
type EventHeader struct {
	PubKeyBLS []byte
	Round     uint64
	Step      uint8
}

// Equal as specified in the Event interface
func (a *EventHeader) Equal(e wire.Event) bool {
	other, ok := e.(*ConsensusEvent)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

// EventHeaderMarshaller marshals a consensus EventHeader as follows:
// - BLS Public Key
// - Round
// - Step
type EventHeaderMarshaller struct{}

// Marshal an EventHeader into a Buffer
func (ehm *EventHeaderMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*Event)

	if err := encoding.WriteVarBytes(buffer, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, consensusEv.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(buffer, consensusEv.Step); err != nil {
		return err
	}

	return nil
}

// EventHeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
type EventHeaderUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

// Unmarshal unmarshals the buffer into a ConsensusEvent
func (a *EventHeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	consensusEv := ev.(*Event)

	// Decoding PubKey BLS
	if err := encoding.ReadVarBytes(r, &consensusEv.PubKeyBLS); err != nil {
		return err
	}

	// Decoding Round
	if err := encoding.ReadUint64(r, binary.LittleEndian, &ConsensusEvent.Round); err != nil {
		return err
	}

	// Decoding Step
	if err := encoding.ReadUint8(r, &committeeEvent.Step); err != nil {
		return err
	}

	return nil
}

// Collector is a helper that groups common operations performed on Events related to a committee
type Collector struct {
	wire.StepEventCollector
	committee    user.Committee
	currentRound uint64
}

// ShouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (cc *Collector) ShouldBeSkipped(m *ConsensusEvent) bool {
	isDupe := cc.Contains(m, m.Step)
	isPleb := !cc.committee.IsMember(m.PubKeyBLS)
	//TODO: the round element needs to be reassessed
	err := cc.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
	failedVerification := err != nil
	return isDupe || isPleb || failedVerification
}
