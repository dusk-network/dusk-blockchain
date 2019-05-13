package agreement

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

type agreementHandler struct {
	committee.Committee
	*events.AggregatedAgreementUnMarshaller
}

func newHandler(committee committee.Committee) *agreementHandler {
	return &agreementHandler{
		Committee:                       committee,
		AggregatedAgreementUnMarshaller: events.NewAggregatedAgreementUnMarshaller(),
	}
}

func (a *agreementHandler) ExtractHeader(e wire.Event) *events.Header {
	ev := e.(*events.AggregatedAgreement)
	return &events.Header{
		Round: ev.Round,
	}
}

func (a *agreementHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*events.AggregatedAgreement)
	return encoding.WriteUint8(r, ev.Step)
}

// Verify checks the signature of the set. TODO: At the moment the overall BLS signature is not checked as it is not clear if checking the ED25519 is enough (it should be in case the node links the BLS keys to the Edward keys)
func (a *agreementHandler) Verify(e wire.Event) error {
	ev, ok := e.(*events.AggregatedAgreement)
	if !ok {
		return errors.New("Cant' verify an event different than the aggregated agreement")
	}
	allVoters := 0
	for i, votes := range ev.VotesPerStep {
		step := ev.Step + uint8(i-1) // the event step is the second one of the reduction cycle
		subcommittee := a.Unpack(votes.BitSet, ev.Round, step)
		allVoters += len(subcommittee)

		apk, err := ReconstructApk(subcommittee)
		if err != nil {
			return err
		}

		signed := new(bytes.Buffer)

		// TODO: change into Header
		vote := &events.Vote{
			Round:     ev.Round,
			Step:      step,
			BlockHash: ev.BlockHash,
		}

		if err := events.MarshalSignableVote(signed, vote); err != nil {
			return err
		}

		if err := bls.Verify(apk, signed.Bytes(), votes.Signature); err != nil {
			return err
		}
	}

	if allVoters < a.Quorum() {
		return errors.New("vote set too small")
	}
	return nil
}

func ReconstructApk(subcommittee sortedset.Set) (*bls.Apk, error) {
	var apk *bls.Apk
	if len(subcommittee) == 0 {
		return nil, errors.New("Subcommittee is empty")
	}
	for i, ipk := range subcommittee {
		pk, err := bls.UnmarshalPk(ipk.Bytes())
		if err != nil {
			return nil, err
		}
		if i == 0 {
			apk = bls.NewApk(pk)
			continue
		}
		if err := apk.Aggregate(pk); err != nil {
			return nil, err
		}
	}

	return apk, nil
}
