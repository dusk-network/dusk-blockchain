package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
	"golang.org/x/crypto/ed25519"
)

type agreementSigner struct {
	*eventSigner
	*events.OutgoingAgreementUnmarshaller
	committee.Committee
}

func NewAgreementSigner(committee committee.Committee, keys *user.Keys) *agreementSigner {
	return &agreementSigner{
		Committee:                     committee,
		eventSigner:                   newEventSigner(keys),
		OutgoingAgreementUnmarshaller: events.NewOutgoingAgreementUnmarshaller(),
	}
}

func (as *agreementSigner) Sign(buf *bytes.Buffer) error {
	a := events.NewAgreement()
	if err := as.Unmarshal(buf, a); err != nil {
		return err
	}

	a.Header.PubKeyBLS = as.BLSPubKey.Marshal()
	aggregatedEv, err := as.Aggregate(a)
	if err != nil {
		return err
	}

	sig, err := as.SignVotes(aggregatedEv.VotesPerStep)
	if err != nil {
		return err
	}
	aggregatedEv.SignedVotes = sig

	buffer, err := events.MarshalAggregatedAgreement(aggregatedEv)
	if err != nil {
		return err
	}

	message := as.signEd25519(buffer)
	*buf = *message
	return nil
}

func (as *agreementSigner) SignVotes(votes []*events.StepVotes) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := events.MarshalVotes(buffer, votes); err != nil {
		return nil, err
	}

	signedVoteSet, err := bls.Sign(as.BLSSecretKey, as.BLSPubKey, buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return signedVoteSet.Compress(), nil
}

func (as *agreementSigner) signEd25519(eventBuf *bytes.Buffer) *bytes.Buffer {
	signature := ed25519.Sign(*as.EdSecretKey, eventBuf.Bytes())
	buf := new(bytes.Buffer)
	if err := encoding.Write512(buf, signature); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, as.EdPubKeyBytes()); err != nil {
		panic(err)
	}

	if _, err := buf.Write(eventBuf.Bytes()); err != nil {
		panic(err)
	}

	return buf
}

// Aggregate the Agreement event into an AggregatedAgreement outgoing event
func (as *agreementSigner) Aggregate(a *events.Agreement) (*events.AggregatedAgreement, error) {
	stepVotesMap := make(map[uint8]struct {
		*events.StepVotes
		sortedset.Set
	})

	for _, ev := range a.VoteSet {
		reduction := ev.(*events.Reduction)
		sv, found := stepVotesMap[reduction.Step]
		if !found {
			sv.StepVotes = events.NewStepVotes()
			sv.Set = sortedset.New()
		}

		if err := sv.StepVotes.Add(reduction); err != nil {
			return nil, err
		}
		sv.Set.Insert(reduction.PubKeyBLS)
		stepVotesMap[reduction.Step] = sv
	}

	aggregatedAgreement := events.NewAggregatedAgreement(a)
	i := 0
	for _, stepVotes := range stepVotesMap {
		sv, provisioners := stepVotes.StepVotes, stepVotes.Set
		sv.BitSet = as.Pack(provisioners, a.Round, sv.Step)
		aggregatedAgreement.VotesPerStep[i] = sv
		i++
	}

	return aggregatedAgreement, nil
}
