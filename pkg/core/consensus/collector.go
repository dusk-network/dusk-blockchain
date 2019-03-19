package consensus

import (
	"bytes"
	"encoding/binary"

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
	other, ok := e.(*EventHeader)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

// EventHeaderMarshaller marshals a consensus EventHeader as follows:
// - BLS Public Key
// - Round
// - Step
type EventHeaderMarshaller struct{}

// Marshal an EventHeader into a Buffer
func (ehm *EventHeaderMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*EventHeader)

	if err := encoding.WriteVarBytes(r, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, consensusEv.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, consensusEv.Step); err != nil {
		return err
	}

	return nil
}

// EventHeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
type EventHeaderUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

func NewEventHeaderUnmarshaller(validate func(*bytes.Buffer) error) *EventHeaderUnmarshaller {
	return &EventHeaderUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a ConsensusEvent
func (a *EventHeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	consensusEv := ev.(*EventHeader)

	// Decoding PubKey BLS
	if err := encoding.ReadVarBytes(r, &consensusEv.PubKeyBLS); err != nil {
		return err
	}

	// Decoding Round
	if err := encoding.ReadUint64(r, binary.LittleEndian, &consensusEv.Round); err != nil {
		return err
	}

	// Decoding Step
	if err := encoding.ReadUint8(r, &consensusEv.Step); err != nil {
		return err
	}

	return nil
}
