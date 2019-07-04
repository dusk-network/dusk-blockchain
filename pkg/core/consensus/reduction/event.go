package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

type (
	// Reduction represents a provisioner vote during the Block Reduction phase of
	// the consensus.
	Reduction struct {
		*header.Header
		SignedHash []byte
	}

	// UnMarshaller marshals and unmarshals Reduction events.
	UnMarshaller struct {
		*header.UnMarshaller
	}
)

// New returns and empty Reduction event.
func New() *Reduction {
	return &Reduction{
		Header: &header.Header{},
	}
}

// Equal as specified in the Event interface
func (e *Reduction) Equal(ev wire.Event) bool {
	other, ok := ev.(*Reduction)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

// NewUnMarshaller returns an initialized UnMarshaller.
func NewUnMarshaller() *UnMarshaller {
	return &UnMarshaller{header.NewUnMarshaller()}
}

// Deserialize a Reduction event from a buffer to its struct representation.
func (u *UnMarshaller) Deserialize(b *bytes.Buffer) (wire.Event, error) {
	ev := New()
	if err := u.Unmarshal(b, ev); err != nil {
		return nil, err
	}

	return ev, nil
}

// Unmarshal unmarshals the buffer into a Reduction event.
func (u *UnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	if err := u.UnMarshaller.Unmarshal(r, bev.Header); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// Marshal a Reduction event into a buffer.
func (u *UnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	if err := u.UnMarshaller.Marshal(r, bev.Header); err != nil {
		return err
	}

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

// SignBuffer is a shortcut to BLS and ED25519 sign a reduction message
func SignBuffer(buf *bytes.Buffer, keys user.Keys) error {
	e := New()
	if err := header.UnmarshalSignableVote(buf, e.Header); err != nil {
		return err
	}

	signed, err := Sign(e, keys)
	if err != nil {
		return err
	}

	*buf = *signed
	return nil
}

// Sign will add the BLS and Ed25519 signatures to a Reduction event.
func Sign(e *Reduction, keys user.Keys) (*bytes.Buffer, error) {
	if err := BlsSign(e, keys); err != nil {
		return nil, err
	}

	outbuf := new(bytes.Buffer)
	unMarshaller := NewUnMarshaller()
	if err := unMarshaller.Marshal(outbuf, e); err != nil {
		return nil, err
	}

	signature := ed25519.Sign(*keys.EdSecretKey, outbuf.Bytes())
	signed := new(bytes.Buffer)
	if err := encoding.Write512(signed, signature); err != nil {
		return nil, err
	}

	if err := encoding.Write256(signed, keys.EdPubKeyBytes); err != nil {
		return nil, err
	}

	if _, err := signed.Write(outbuf.Bytes()); err != nil {
		return nil, err
	}

	return signed, nil
}

// BlsSign is a shortcut to create a BLS signature of a reduction vote and fill the
// proper field in Reduction struct
func BlsSign(ev *Reduction, keys user.Keys) error {
	buf := new(bytes.Buffer)

	if err := header.MarshalSignableVote(buf, ev.Header); err != nil {
		return err
	}

	signedHash, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, buf.Bytes())
	if err != nil {
		return err
	}
	ev.SignedHash = signedHash.Compress()
	ev.PubKeyBLS = keys.BLSPubKeyBytes
	return nil
}
