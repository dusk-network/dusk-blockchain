package header

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

type (
	// Header is an embeddable struct representing the consensus event header fields
	Header struct {
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
		BlockHash []byte
	}
)

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
func Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*Header)
	if err := encoding.WriteVarBytes(r, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	return MarshalSignableVote(r, consensusEv)
}

// Unmarshal unmarshals the buffer into a Header.
func Unmarshal(r *bytes.Buffer, ev wire.Event) error {
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
	if err := encoding.WriteUint64LE(r, vote.Round); err != nil {
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
	if err := encoding.ReadUint64LE(r, &vote.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &vote.Step); err != nil {
		return err
	}

	vote.BlockHash = make([]byte, 32)
	return encoding.Read256(r, vote.BlockHash)
}
