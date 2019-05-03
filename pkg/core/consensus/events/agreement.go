package events

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// StepVotes represents the aggregated votes for one reduction step. Normally an Agreement event includes two of these structures. They need to be kept separated since the BitSet representation of the Signees does not admit duplicates, whereas the same provisioner may very well be included in the committee for both Reduction steps
	StepVotes struct {
		Apk       []byte
		BitSet    uint64
		Signature []byte
	}

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

	OutgoingAgreementUnmarshaller struct {
		ReductionUnmarshaller
	}
)

// NewStepVotes returns a new StepVotes structure for a given round, step and block hash
func NewStepVotes(apk []byte, bitset uint64, signature []byte) *StepVotes {
	return &StepVotes{
		Apk:       apk,
		BitSet:    bitset,
		Signature: signature,
	}
}

func (sv *StepVotes) Equal(other *StepVotes) bool {
	return bytes.Equal(sv.Apk, other.Apk) && bytes.Equal(sv.Signature, other.Signature)
}

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

// UnmarshalVotes unmarshals the array of StepVotes for a single Agreement
func UnmarshalVotes(r *bytes.Buffer, votes *[]*StepVotes) error {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	for i := uint64(0); i < length; i++ {
		sv, err := UnmarshalStepVotes(r)
		if err != nil {
			return err
		}

		(*votes)[i] = sv
	}

	return nil
}

// UnmarshalStepVotes unmarshals a single StepVote
func UnmarshalStepVotes(r *bytes.Buffer) (*StepVotes, error) {
	// APK
	apk := make([]byte, 129)
	if err := encoding.ReadVarBytes(r, &apk); err != nil {
		return nil, err
	}

	// BitSet
	bitset := uint64(0)
	if err := encoding.ReadUint64(r, binary.LittleEndian, &bitset); err != nil {
		return nil, err
	}

	// Signature
	signature := make([]byte, 33)
	if err := encoding.ReadBLS(r, &signature); err != nil {
		return nil, err
	}

	return NewStepVotes(apk, bitset, signature), nil
}

// MarshalVotes marshals an array of StepVotes
func MarshalVotes(r *bytes.Buffer, votes []*StepVotes) error {
	if err := encoding.WriteVarInt(r, uint64(len(votes))); err != nil {
		return err
	}

	for _, stepVotes := range votes {
		if err := MarshalStepVotes(r, stepVotes); err != nil {
			return err
		}
	}

	return nil
}

// MarshalStepVotes marshals the aggregated form of the BLS PublicKey and Signature for an ordered set of votes
func MarshalStepVotes(r *bytes.Buffer, vote *StepVotes) error {
	// APK
	if err := encoding.WriteVarBytes(r, vote.Apk); err != nil {
		return err
	}

	// BitSet
	if err := encoding.WriteUint64(r, binary.LittleEndian, vote.BitSet); err != nil {
		return err
	}

	// Signature
	if err := encoding.WriteBLS(r, vote.Signature); err != nil {
		return err
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
	return nil
}

func NewOutgoingAgreementUnmarshaller() *OutgoingAgreementUnmarshaller {
	return &OutgoingAgreementUnmarshaller{
		ReductionUnmarshaller: NewAgreementUnMarshaller(),
	}
}

func (ceu *OutgoingAgreementUnmarshaller) NewEvent() wire.Event {
	return NewAgreement()
}

func (ceu *OutgoingAgreementUnmarshaller) Unmarshal(agreementBuffer *bytes.Buffer, ev wire.Event) error {
	aev := ev.(*Agreement)
	if err := encoding.ReadUint64(agreementBuffer, binary.LittleEndian, &aev.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(agreementBuffer, &aev.Step); err != nil {
		return err
	}

	if err := encoding.Read256(agreementBuffer, &aev.AgreedHash); err != nil {
		return err
	}

	voteSet, err := ceu.UnmarshalVoteSet(agreementBuffer)
	if err != nil {
		return err
	}

	aev.VoteSet = voteSet

	return nil
}
