package events

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// Agreement is the message that encapsulates data relevant for
	// components relying on committee information
	Agreement struct {
		*Header
		VoteSet       []wire.Event
		SignedVoteSet []byte
		AgreedHash    []byte
	}
	// AgreementUnMarshaller implements both Marshaller and Unmarshaller interface
	AgreementUnMarshaller struct {
		*UnMarshaller
		ReductionUnmarshaller
	}
)

// NewAgreement returns an empty Agreement event.
func NewAgreement() *Agreement {
	return &Agreement{
		Header: &Header{},
	}
}

// Equal as specified in the Event interface
func (ceh *Agreement) Equal(e wire.Event) bool {
	other, ok := e.(*Agreement)
	return ok && ceh.Header.Equal(other.Header) &&
		bytes.Equal(other.SignedVoteSet, ceh.SignedVoteSet)
}

// NewAgreementUnMarshaller creates a new AgreementUnMarshaller. Internally it creates an HeaderUnMarshaller which takes care of Decoding and Encoding operations
func NewAgreementUnMarshaller() *AgreementUnMarshaller {

	return &AgreementUnMarshaller{
		ReductionUnmarshaller: NewReductionUnMarshaller(),
		UnMarshaller:          NewUnMarshaller(),
	}
}

func (a *AgreementUnMarshaller) NewEvent() wire.Event {
	return NewAgreement()
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
	return nil
}
