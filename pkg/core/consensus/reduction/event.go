package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

type (
	// Reduction is a basic reduction event.
	Reduction struct {
		*events.Header
		SignedHash []byte
	}

	// ReductionUnMarshaller marshals and unmarshales th
	ReductionUnMarshaller struct {
		*events.UnMarshaller
	}

	ReductionUnmarshaller interface {
		wire.EventMarshaller
		wire.EventDeserializer
	}
)

// NewReduction returns and empty Reduction event.
func NewReduction() *Reduction {
	return &Reduction{
		Header: &events.Header{},
	}
}

// Equal as specified in the Event interface
func (e *Reduction) Equal(ev wire.Event) bool {
	other, ok := ev.(*Reduction)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

func NewReductionUnMarshaller() *ReductionUnMarshaller {
	return &ReductionUnMarshaller{events.NewUnMarshaller()}
}

func (r *ReductionUnMarshaller) Deserialize(b *bytes.Buffer) (wire.Event, error) {
	ev := NewReduction()
	if err := r.Unmarshal(b, ev); err != nil {
		return nil, err
	}

	return ev, nil
}

// Unmarshal unmarshals the buffer into a Committee
func (a *ReductionUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	if err := a.HeaderUnmarshaller.Unmarshal(r, bev.Header); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// Marshal a Reduction into a buffer.
func (a *ReductionUnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	if err := a.HeaderMarshaller.Marshal(r, bev.Header); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// TODO: generalize the set/array marshalling/unmarshalling through an interface
func (a *ReductionUnMarshaller) UnmarshalVoteSet(r *bytes.Buffer) ([]wire.Event, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]wire.Event, length)
	for i := uint64(0); i < length; i++ {
		rev := &Reduction{
			Header: &events.Header{},
		}
		if err := a.Unmarshal(r, rev); err != nil {
			return nil, err
		}

		evs[i] = rev
	}

	return evs, nil
}

func (a *ReductionUnMarshaller) MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := a.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

// SignReduction is a shortcut to BLS and ED25519 sign a reduction message
func SignReduction(buf *bytes.Buffer, keys *user.Keys) error {
	e := NewReduction()
	if err := events.UnmarshalSignableVote(buf, e.Header); err != nil {
		return err
	}

	signed, err := SignReductionEvent(e, keys)
	if err != nil {
		return err
	}

	*buf = *signed
	return nil
}

func SignReductionEvent(e *Reduction, keys *user.Keys) (*bytes.Buffer, error) {
	if err := BlsSignReductionEvent(e, keys); err != nil {
		return nil, err
	}
	outbuf := new(bytes.Buffer)

	unMarshaller := NewReductionUnMarshaller()
	if err := unMarshaller.Marshal(outbuf, e); err != nil {
		return nil, err
	}

	signature := ed25519.Sign(*keys.EdSecretKey, outbuf.Bytes())

	signed := new(bytes.Buffer)
	if err := encoding.Write512(signed, signature); err != nil {
		return nil, err
	}

	edPubKeyBuf := new(bytes.Buffer)
	if err := encoding.Write512(edPubKeyBuf, signature); err != nil {
		return nil, err
	}

	if _, err := signed.Write(outbuf.Bytes()); err != nil {
		return nil, err
	}

	return signed, nil
}

// BlsSignReductionEvent is a shortcut to create a BLS signature of a reduction vote and fill the proper field in Reduction struct
func BlsSignReductionEvent(ev *Reduction, keys *user.Keys) error {
	buf := new(bytes.Buffer)

	if err := events.MarshalSignableVote(buf, ev.Header); err != nil {
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
