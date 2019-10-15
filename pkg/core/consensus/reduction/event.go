package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

type (
	// Reduction represents a provisioner vote during the Block Reduction phase of
	// the consensus.
	Reduction struct {
		SignedHash []byte
	}
)

// New returns and empty Reduction event.
func New() *Reduction {
	return &Reduction{
		SignedHash: make([]byte, 33),
	}
}

// Unmarshal unmarshals the buffer into a Reduction event.
func Unmarshal(r *bytes.Buffer, bev *Reduction) error {
	bev.SignedHash = make([]byte, 33)
	if err := encoding.ReadBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// Marshal a Reduction event into a buffer.
func Marshal(r *bytes.Buffer, bev Reduction) error {
	if err := encoding.WriteBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

func UnmarshalVoteSet(r *bytes.Buffer) ([]Reduction, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]Reduction, length)
	for i := uint64(0); i < length; i++ {
		rev := New()
		if err := Unmarshal(r, rev); err != nil {
			return nil, err
		}

		evs[i] = *rev
	}

	return evs, nil
}

// MarshalVoteSet marshals a slice of Reduction events to a buffer.
func MarshalVoteSet(r *bytes.Buffer, evs []Reduction) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

// TODO: sign in the consensus.Consensus
// SignBuffer is a shortcut to BLS
/*
func SignBuffer(buf *bytes.Buffer, keys user.Keys) error {
	e := New()
	if err := Unmarshal(buf, e); err != nil {
		return err
	}

	if err := BlsSign(e, keys); err != nil {
		return nil, err
	}

	*buf = *signed
	return nil
}
*/
