package header

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

	// headerMarshaller marshals a consensus Header as follows:
	// - BLS Public Key
	// - Round
	// - Step
	headerMarshaller struct{}

	// headerUnmarshaller unmarshals consensus headers.
	headerUnmarshaller struct{}

	// UnMarshaller marshals and unmarshals consensus event headers. It is a helper
	// to be embedded in the various consensus message unmarshallers.
	UnMarshaller struct {
		*headerMarshaller
		*headerUnmarshaller
	}
)

// NewUnMarshaller instantiates a struct to Marshal and Unmarshal event Headers
func NewUnMarshaller() *UnMarshaller {
	return &UnMarshaller{
		headerMarshaller:   new(headerMarshaller),
		headerUnmarshaller: new(headerUnmarshaller),
	}
}

// Sender implements wire.Event.
// Returns the BLS public key of the event sender.
func (h *Header) Sender() []byte {
	return h.PubKeyBLS
}

// Equal implements wire.Event.
// Checks if two headers are the same.
func (h *Header) Equal(e wire.Event) bool {
	other, ok := e.(*Header)
	return ok && (bytes.Equal(h.PubKeyBLS, other.PubKeyBLS)) &&
		(h.Round == other.Round) && (h.Step == other.Step) &&
		(bytes.Equal(h.BlockHash, other.BlockHash))
}

// Marshal a Header into a Buffer.
func (hm *headerMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*Header)
	if err := encoding.WriteVarBytes(r, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	return MarshalSignableVote(r, consensusEv)
}

// Unmarshal unmarshals the buffer into a Header.
func (hu *headerUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*Header)

	// Decoding PubKey BLS
	if err := encoding.ReadVarBytes(r, &consensusEv.PubKeyBLS); err != nil {
		return err
	}

	return UnmarshalSignableVote(r, consensusEv)
}

// MarshalSignableVote marshals the fields necessary for a Committee member to cast
// a Vote (namely the Round, the Step and the BlockHash).
func MarshalSignableVote(r *bytes.Buffer, vote *Header) error {
	if err := encoding.WriteUint64(r, binary.LittleEndian, vote.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, vote.Step); err != nil {
		return err
	}

	return encoding.Write256(r, vote.BlockHash)
}

// UnmarshalSignableVote unmarshals the fields necessary for a Committee member to cast
// a Vote (namely the Round, the Step and the BlockHash).
func UnmarshalSignableVote(r *bytes.Buffer, vote *Header) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &vote.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &vote.Step); err != nil {
		return err
	}

	return encoding.Read256(r, &vote.BlockHash)
}
