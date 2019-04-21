package agreement

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/hashset"
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

func (a *agreementHandler) NewEvent() wire.Event {
	return events.NewAgreement()
}

func (a *agreementHandler) ExtractHeader(e wire.Event) *events.Header {
	ev := e.(*events.Agreement)
	return &events.Header{
		Round: ev.Round,
		Step:  ev.Step,
	}
}

func (a *agreementHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*events.Agreement)
	return encoding.WriteUint8(r, ev.Step)
}

// Verify checks the signature of the set
func (a *agreementHandler) Verify(e wire.Event) error {
	ev := e.(*events.Agreement)
	return a.verify(ev.VoteSet, ev.AgreedHash, ev.Round, ev.Step)
}

func (a *agreementHandler) verify(voteSet []wire.Event, hash []byte, round uint64, step uint8) error {

	votes := 0
	voted := make(map[uint8]*hashset.Set)

	for _, ev := range voteSet {
		vote := ev.(*events.Reduction)
		voters, found := voted[vote.Step]
		if !found {
			voters = hashset.New()
		}
		if voters.Add(vote.Sender()) {
			// if the voter has already voted for this step, we ignore the vote but continue
			// TODO: we should probably do something with this nodes' reputation
			continue
		}

		if !fromValidStep(vote.Step, step) {
			return errors.New("vote does not belong to vote set")
		}

		if !a.IsMember(vote.Sender(), round, vote.Step) {
			return errors.New("voter is not eligible to vote")
		}

		if err := msg.VerifyBLSSignature(vote.PubKeyBLS, vote.VotedHash,
			vote.SignedHash); err != nil {

			return errors.New("BLS verification failed")
		}

		votes++
		if votes >= a.Quorum() {
			// as soon as we see a quorum for a step we return
			return nil
		}
		// if the voter never voted for the step we add her to the set
		voted[vote.Step] = voters
	}

	// not enough votes collected
	return errors.New("vote set too small")
}

func fromValidStep(voteStep, setStep uint8) bool {
	return voteStep == setStep || voteStep+1 == setStep
}
