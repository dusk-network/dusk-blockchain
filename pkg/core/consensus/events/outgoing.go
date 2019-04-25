package events

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	OutgoingReductionUnmarshaller struct {
		ReductionUnmarshaller
	}

	OutgoingAgreementUnmarshaller struct {
		ReductionUnmarshaller
	}
)

func NewOutgoingReductionUnmarshaller() *OutgoingReductionUnmarshaller {
	return &OutgoingReductionUnmarshaller{
		ReductionUnmarshaller: NewReductionUnMarshaller(),
	}
}

func (a *OutgoingReductionUnmarshaller) NewEvent() wire.Event {
	return NewReduction()
}

func (a *OutgoingReductionUnmarshaller) Unmarshal(reductionBuffer *bytes.Buffer, ev wire.Event) error {
	rev := ev.(*Reduction)
	if err := encoding.ReadUint64(reductionBuffer, binary.LittleEndian, &rev.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(reductionBuffer, &rev.Step); err != nil {
		return err
	}

	if err := encoding.Read256(reductionBuffer, &rev.VotedHash); err != nil {
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
