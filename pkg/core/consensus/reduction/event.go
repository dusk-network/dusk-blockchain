package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
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

// Equal as specified in the Event interface
func (e *Reduction) Equal(ev wire.Event) bool {
	other, ok := ev.(*Reduction)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

// Deserialize a Reduction event from a buffer to its struct representation.
func Deserialize(b *bytes.Buffer) (wire.Event, error) {
	ev := New()
	if err := Unmarshal(b, ev); err != nil {
		return nil, err
	}

	return ev, nil
}

// Unmarshal unmarshals the buffer into a Reduction event.
func Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	bev.SignedHash = make([]byte, 33)
	if err = encoding.ReadBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// Marshal a Reduction event into a buffer.
func (u *UnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	if err := encoding.WriteBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// UnmarshalVoteSet unmarshals a slice of Reduction events from a buffer.
// TODO: generalize the set/array marshalling/unmarshalling through an interface
func (u *UnMarshaller) UnmarshalVoteSet(r *bytes.Buffer) ([]wire.Event, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]wire.Event, length)
	for i := uint64(0); i < length; i++ {
		rev, err := u.Deserialize(r)
		if err != nil {
			return nil, err
		}

		evs[i] = rev
	}

	return evs, nil
}

// MarshalVoteSet marshals a slice of Reduction events to a buffer.
func (u *UnMarshaller) MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := u.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

// TODO: sign in the consensus.Consensus
// SignBuffer is a shortcut to BLS and ED25519 sign a reduction message
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
