package generation

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type certificateGenerator struct {
	agreementEvent wire.Event
}

func (a *certificateGenerator) setAgreementEvent(m *bytes.Buffer) error {
	unmarshaller := agreement.NewUnMarshaller()
	ev, err := unmarshaller.Deserialize(m)
	if err != nil {
		return err
	}

	a.agreementEvent = ev
	return nil
}

func (a *certificateGenerator) isReady(round uint64) bool {
	ev, ok := a.agreementEvent.(*agreement.Agreement)
	if !ok {
		// Prevent panics when just starting
		return false
	}

	return ev.Round == round
}

func (a *certificateGenerator) generateCertificate() *block.Certificate {
	ev := a.agreementEvent.(*agreement.Agreement)
	return &block.Certificate{
		StepOneBatchedSig: ev.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: ev.VotesPerStep[1].Signature.Compress(),
		Step:              ev.Step,
		StepOneCommittee:  ev.VotesPerStep[0].BitSet,
		StepTwoCommittee:  ev.VotesPerStep[1].BitSet,
	}
}
