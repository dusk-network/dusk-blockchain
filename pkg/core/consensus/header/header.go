package header

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/bls"
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

// Writer writes the header to a Buffer. It is an interface injected into components
type Writer interface {
	WriteHeader([]byte, *bytes.Buffer) error
}

// Phase is used to introduce a time order to the Header
type Phase uint8

const (
	// Same indicates that headers belong to the same phase
	Same Phase = iota
	// Before indicates that the header indicates a past event
	Before
	// After indicates that the header indicates a future event
	After
)

// Sender implements wire.Event.
// Returns the BLS public key of the event sender.
func (h Header) Sender() []byte {
	return h.PubKeyBLS
}

// Compare headers to establish time order
func (h Header) CompareRoundAndStep(round uint64, step uint8) Phase {
	comparison := h.CompareRound(round)
	if comparison == Same {
		if h.Step < step {
			return Before
		}

		if h.Step == step {
			return Same
		}

		return After
	}

	return comparison
}

func (h Header) CompareRound(round uint64) Phase {
	if h.Round < round {
		return Before
	}

	if h.Round == round {
		return Same
	}

	return After
}

// Equal implements wire.Event.
// Checks if two headers are the same.
func (h Header) Equal(e wire.Event) bool {
	other, ok := e.(Header)
	return ok && (bytes.Equal(h.PubKeyBLS, other.PubKeyBLS)) &&
		(h.Round == other.Round) && (h.Step == other.Step) &&
		(bytes.Equal(h.BlockHash, other.BlockHash))
}

// Marshal a Header into a Buffer.
func Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(Header)
	if err := encoding.WriteVarBytes(r, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	return MarshalFields(r, consensusEv)
}

// Compose is useful when header information is cached and there is an opportunity to avoid unnecessary allocations
func Compose(blsPubKey bytes.Buffer, phase bytes.Buffer, hash []byte) (bytes.Buffer, error) {
	if _, err := blsPubKey.ReadFrom(&phase); err != nil {
		return bytes.Buffer{}, err
	}

	if err := encoding.Write256(&blsPubKey, hash); err != nil {
		return bytes.Buffer{}, err
	}

	return blsPubKey, nil
}

// Unmarshal unmarshals the buffer into a Header.
func Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*Header)

	// Decoding PubKey BLS
	if err := encoding.ReadVarBytes(r, &consensusEv.PubKeyBLS); err != nil {
		return err
	}

	return UnmarshalFields(r, consensusEv)
}

// MarshalFields marshals the core field of the Header (i.e. Round, Step and BlockHash)
func MarshalFields(r *bytes.Buffer, h Header) error {
	if err := encoding.WriteUint64LE(r, h.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, h.Step); err != nil {
		return err
	}

	return encoding.Write256(r, h.BlockHash)
}

// UnmarshalFields unmarshals the core field of the Header (i.e. Round, Step and BlockHash)
func UnmarshalFields(r *bytes.Buffer, h *Header) error {
	if err := encoding.ReadUint64LE(r, &h.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &h.Step); err != nil {
		return err
	}

	h.BlockHash = make([]byte, 32)
	return encoding.Read256(r, h.BlockHash)
}

// MarshalSignableVote marshals the fields necessary for a Committee member to cast
// a Vote (namely the Round, the Step and the BlockHash).
// Note: UnmarshalSignableVote does not make sense as the only reason to use it would be if we could somehow revert a signature to the preimage and thus unmarshal it into a struct :P
func MarshalSignableVote(r *bytes.Buffer, h Header) error {
	return MarshalFields(r, h)
}

// VerifySignatures verifies the BLS aggregated signature carried by consensus related messages.
// The signed message needs to carry information about the round, the step and the blockhash
func VerifySignatures(round uint64, step uint8, blockHash []byte, apk *bls.Apk, sig *bls.Signature) error {
	signed := new(bytes.Buffer)
	vote := Header{
		Round:     round,
		Step:      step,
		BlockHash: blockHash,
	}

	if err := MarshalSignableVote(signed, vote); err != nil {
		return err
	}

	return bls.Verify(apk, signed.Bytes(), sig)
}
