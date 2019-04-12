package consensus

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// EventHeader is an embeddable struct representing the consensus event header fields
	EventHeader struct {
		// TODO: ed25519 related fields added for demo to facilitate easy
		// re-propagation. review
		Signature []byte
		PubKeyEd  []byte
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
	}

	// EventHeaderMarshaller marshals a consensus EventHeader as follows:
	// - BLS Public Key
	// - Round
	// - Step
	EventHeaderMarshaller struct{}

	// EventHeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	EventHeaderUnmarshaller struct {
		Validate func([]byte, []byte, []byte) error
	}
)

// Equal as specified in the Event interface
func (a *EventHeader) Equal(e wire.Event) bool {
	other, ok := e.(*EventHeader)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

// Sender of the Event
func (a *EventHeader) Sender() []byte {
	return a.PubKeyBLS
}

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

func (ehm *EventHeaderMarshaller) MarshalEdFields(r *bytes.Buffer, ev wire.Event) error {
	evh := ev.(*EventHeader)
	if err := encoding.Write512(r, evh.Signature); err != nil {
		return err
	}

	if err := encoding.Write256(r, evh.PubKeyEd); err != nil {
		return err
	}

	return nil
}

// NewEventHeaderUnmarshaller creates an EventHeaderUnmarshaller delegating validation to the validate function
func NewEventHeaderUnmarshaller(validate func([]byte, []byte, []byte) error) *EventHeaderUnmarshaller {
	return &EventHeaderUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a ConsensusEvent
func (a *EventHeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// if the injection is unsuccessful, panic
	consensusEv := ev.(*EventHeader)
	if err := encoding.Read512(r, &consensusEv.Signature); err != nil {
		return err
	}

	if err := encoding.Read256(r, &consensusEv.PubKeyEd); err != nil {
		return err
	}

	// verify the signature here
	if err := a.Validate(consensusEv.PubKeyEd, r.Bytes(), consensusEv.Signature); err != nil {
		fmt.Println(consensusEv.PubKeyEd)
		fmt.Println(consensusEv.Signature)
		return err
	}

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
