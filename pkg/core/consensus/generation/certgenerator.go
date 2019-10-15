package generation

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
)

type certificateGenerator struct {
	agreementEvent *agreement.Agreement
}

func (a *certificateGenerator) setAgreementEvent(m bytes.Buffer) error {
	hdr := header.Header{}
	if err := header.Unmarshal(&m, &hdr); err != nil {
		return err
	}

	a.agreementEvent = agreement.New(hdr)

	if err := agreement.Unmarshal(&m, a.agreementEvent); err != nil {
		return err
	}

	return nil
}

func (a *certificateGenerator) isReady(round uint64) bool {
	if a.agreementEvent == nil {
		// Prevent panics when just starting
		return false
	}

	return a.agreementEvent.Header.Round == round
}

func (a *certificateGenerator) generateCertificate() *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: a.agreementEvent.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: a.agreementEvent.VotesPerStep[1].Signature.Compress(),
		Step:              a.agreementEvent.Header.Step,
		StepOneCommittee:  a.agreementEvent.VotesPerStep[0].BitSet,
		StepTwoCommittee:  a.agreementEvent.VotesPerStep[1].BitSet,
	}
}
