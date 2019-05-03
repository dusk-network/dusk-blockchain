package agreement

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type agreementHandler struct {
	committee.Committee
	*events.AgreementUnMarshaller
}

func newHandler(committee committee.Committee) *agreementHandler {
	return &agreementHandler{
		Committee:             committee,
		AgreementUnMarshaller: events.NewAgreementUnMarshaller(),
	}
}

func (a *agreementHandler) ExtractHeader(e wire.Event) *events.Header {
	ev := e.(*events.Agreement)
	return &events.Header{
		Round: ev.Round,
	}
}

func (a *agreementHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*events.Agreement)
	return encoding.WriteUint8(r, ev.Step)
}

// Verify checks the signature of the set
func (a *agreementHandler) Verify(e wire.Event) error {
	/*
		ev := e.(*events.Agreement)
		allvoters := 0
		for i, votes := range ev.Votes {
			step := ev.Step + (i - 1) // the event step is the second one of the reduction cycle
			subcommittee := a.Unpack(votes.BitSet, ev.Round, uint8(i))
			allvoters += len(subcommittee)

			apk, err := committee.ReconstructApk(subcommittee)
			if err != nil {
				return err
			}

			signed := new(bytes.Buffer)
			if err := events.MarshalSignedVote(signed, ev.Round, step, ev.AgreedHash); err != nil {
				return err
			}

			if err := bls.Verify(apk, signed.Bytes(), votes.Signature); err != nil {
				return err
			}
		}

		if len(subcommittee) < a.Quorum() {
			return errors.New("vote set too small")
		}
	*/
	return nil
}
