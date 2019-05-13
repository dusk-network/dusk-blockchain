package events

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// Header is an embeddable struct representing the consensus event header fields
	Header struct {
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
		BlockHash []byte
	}

	// HeaderMarshaller marshals a consensus Header as follows:
	// - BLS Public Key
	// - Round
	// - Step
	HeaderMarshaller struct{}

	// HeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	HeaderUnmarshaller struct{}

	UnMarshaller struct {
		*HeaderMarshaller
		*HeaderUnmarshaller
	}
)

// NewUnMarshaller instantiates a struct to Marshal and Unmarshal event Headers
func NewUnMarshaller() *UnMarshaller {
	return &UnMarshaller{
		HeaderMarshaller:   new(HeaderMarshaller),
		HeaderUnmarshaller: new(HeaderUnmarshaller),
	}
}

// Sender of the Event
func (a *Header) Sender() []byte {
	return a.PubKeyBLS
}

// Equal as specified in the Event interface
func (a *Header) Equal(e wire.Event) bool {
	other, ok := e.(*Header)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) &&
		(a.Round == other.Round) && (a.Step == other.Step) && (bytes.Equal(a.BlockHash, other.BlockHash))
}

// Marshal a Header into a Buffer
func (ehm *HeaderMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*Header)
	if err := encoding.WriteVarBytes(r, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, consensusEv.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, consensusEv.Step); err != nil {
		return err
	}

	if err := encoding.Write256(r, consensusEv.BlockHash); err != nil {
		return err
	}

	return nil
}

// Unmarshal unmarshals the buffer into a Consensus
func (a *HeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// if the injection is unsuccessful, panic
	consensusEv := ev.(*Header)

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

	if err := encoding.Read256(r, &consensusEv.BlockHash); err != nil {
		return err
	}

	return nil
}
