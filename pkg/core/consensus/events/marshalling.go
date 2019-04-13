package events

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// HeaderMarshaller marshals a consensus Header as follows:
	// - BLS Public Key
	// - Round
	// - Step
	HeaderMarshaller struct{}

	// HeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	HeaderUnmarshaller struct {
		Validate func([]byte, []byte, []byte) error
	}

	UnMarshaller struct {
		*HeaderMarshaller
		*HeaderUnmarshaller
	}

	ReductionUnMarshaller struct {
		*UnMarshaller
	}

	ReductionUnmarshaller interface {
		wire.EventMarshaller
		wire.EventUnmarshaller
		MarshalVoteSet(*bytes.Buffer, []wire.Event) error
		UnmarshalVoteSet(*bytes.Buffer) ([]wire.Event, error)
	}

	// AgreementUnMarshaller implements both Marshaller and Unmarshaller interface
	AgreementUnMarshaller struct {
		*UnMarshaller
		ReductionUnmarshaller
	}
)

// Marshal an Header into a Buffer
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

	return nil
}

func (ehm *HeaderMarshaller) MarshalEdFields(r *bytes.Buffer, ev wire.Event) error {
	evh := ev.(*Header)
	if err := encoding.Write512(r, evh.Signature); err != nil {
		return err
	}

	if err := encoding.Write256(r, evh.PubKeyEd); err != nil {
		return err
	}

	return nil
}

// NewHeaderUnmarshaller creates an HeaderUnmarshaller delegating validation to the validate function
func NewHeaderUnmarshaller(validate func([]byte, []byte, []byte) error) *HeaderUnmarshaller {
	return &HeaderUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a Consensus
func (a *HeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// if the injection is unsuccessful, panic
	consensusEv := ev.(*Header)
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

func NewUnMarshaller(validate func([]byte, []byte, []byte) error) *UnMarshaller {
	return &UnMarshaller{
		HeaderMarshaller:   new(HeaderMarshaller),
		HeaderUnmarshaller: NewHeaderUnmarshaller(validate),
	}
}

func NewReductionUnMarshaller(validate func([]byte, []byte, []byte) error) *ReductionUnMarshaller {
	return &ReductionUnMarshaller{NewUnMarshaller(validate)}
}

// NewAgreementUnMarshaller creates a new AgreementUnMarshaller. Internally it creates an HeaderUnMarshaller which takes care of Decoding and Encoding operations
func NewAgreementUnMarshaller(validate func([]byte, []byte, []byte) error) *AgreementUnMarshaller {

	return &AgreementUnMarshaller{
		ReductionUnmarshaller: NewReductionUnMarshaller(func([]byte, []byte, []byte) error { return nil }),
		UnMarshaller:          NewUnMarshaller(validate),
	}
}

// Unmarshal unmarshals the buffer into a Committee
func (a *ReductionUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*Reduction)
	if err := a.HeaderUnmarshaller.Unmarshal(r, bev.Header); err != nil {
		return err
	}

	if err := encoding.Read256(r, &bev.VotedHash); err != nil {
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

	if err := encoding.Write256(r, bev.VotedHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

func (a *ReductionUnMarshaller) UnmarshalVoteSet(r *bytes.Buffer) ([]wire.Event, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]wire.Event, length)
	for i := uint64(0); i < length; i++ {
		rev := &Reduction{
			Header: &Header{},
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
		rev := event.(*Reduction)
		if err := a.MarshalEdFields(r, rev.Header); err != nil {
			return err
		}
		if err := a.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

// Unmarshal unmarshals the buffer into a CommitteeHeader
// Field order is the following:
// * Consensus Header [BLS Public Key; Round; Step]
// * Committee Header [Signed Vote Set; Vote Set; BlockHash]
func (ceu *AgreementUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	cev := ev.(*Agreement)
	if err := ceu.HeaderUnmarshaller.Unmarshal(r, cev.Header); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &cev.SignedVoteSet); err != nil {
		return err
	}

	voteSet, err := ceu.UnmarshalVoteSet(r)
	if err != nil {
		return err
	}
	cev.VoteSet = voteSet

	if err := encoding.Read256(r, &cev.AgreedHash); err != nil {
		return err
	}

	return nil
}

// Marshal the buffer into a committee Event
// Field order is the following:
// * Consensus Header [BLS Public Key; Round; Step]
// * Committee Header [Signed Vote Set; Vote Set; BlockHash]
func (ceu *AgreementUnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	// TODO: review
	cev, ok := ev.(*Agreement)
	if !ok {
		// cev is nil
		return nil
	}

	if err := ceu.HeaderMarshaller.Marshal(r, cev.Header); err != nil {
		return err
	}

	// Marshal BLS Signature of VoteSet
	if err := encoding.WriteBLS(r, cev.SignedVoteSet); err != nil {
		return err
	}

	// Marshal VoteSet
	if err := ceu.MarshalVoteSet(r, cev.VoteSet); err != nil {
		return err
	}

	if err := encoding.Write256(r, cev.AgreedHash); err != nil {
		return err
	}
	// TODO: write the vote set to the buffer
	return nil
}
