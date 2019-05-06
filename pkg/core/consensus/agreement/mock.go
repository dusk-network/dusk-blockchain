package agreement

import (
	"bytes"
	"crypto/rand"

	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/voting"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

// PublishMock is a mock-up method to facilitate testing of publishing of Agreement events
func PublishMock(bus wire.EventBroker, hash []byte, round uint64, step uint8, voteNr int) {
	buf := MockAgreementBuf(hash, round, step, voteNr)
	bus.Publish(msg.OutgoingBlockAgreementTopic, buf)
}

func MockAggregatedAgreement(hash []byte, round uint64, step uint8, voteNr int) *events.AggregatedAgreement {
	a := events.NewAggregatedAgreement()
	pk, sk, _ := bls.GenKeyPair(rand.Reader)
	a.AgreedHash = hash
	a.Round = round
	a.Step = step

	a.VotesPerStep = genVotes(hash, round, step, voteNr)
	buf := new(bytes.Buffer)
	_ = events.MarshalVotes(buf, a.VotesPerStep)
	sig, _ := bls.Sign(sk, pk, buf.Bytes())
	a.SignedVotes = sig.Compress()
	a.PubKeyBLS = pk.Marshal()
	return a
}

func genVotes(hash []byte, round uint64, step uint8, nr int) []*events.StepVotes {
	if nr < 2 {
		panic("At least two votes are required to mock an Agreement")
	}
	votes := make([]*events.StepVotes, 2)
	for i := 0; i < nr; i++ {
		keys, _ := user.NewRandKeys()
		stepCycle := i % 2
		stepVote := votes[stepCycle]
		if stepVote == nil {
			stepVote = events.NewStepVotes()
		}
		_, rEv := reduction.MockReduction(keys, hash, round, step-uint8(stepCycle))
		if err := stepVote.Add(rEv); err != nil {
			panic(err)
		}
		votes[stepCycle] = stepVote
	}
	return votes
}

func MockAgreement(keys *user.Keys, hash []byte, round uint64, step uint8, votes []wire.Event) *events.Agreement {
	ev := events.NewAgreement()
	ev.Round = round
	ev.Step = step
	ev.PubKeyBLS = keys.BLSPubKeyBytes
	ev.VoteSet = votes
	ev.AgreedHash = hash
	return ev
}

func MockVoteAgreement(hash []byte, round uint64, step uint8, voteNr int) wire.Event {
	var k *user.Keys
	var e wire.Event
	votes := make([]wire.Event, 0)
	for i := 0; i < voteNr; i++ {
		if i == 0 {
			k, e = reduction.MockReduction(nil, hash, round, step)
			votes = append(votes, e)
			continue
		}
		_, e = reduction.MockReduction(nil, hash, round, step)
		votes = append(votes, e)
	}
	ev := MockAgreement(k, hash, round, step, votes)
	return ev
}

func MockAgreementBuf(hash []byte, round uint64, step uint8, voteNr int) *bytes.Buffer {
	ev := MockVoteAgreement(hash, round, step, voteNr)
	signer := voting.NewAgreementSigner(nil, nil)
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	_ = signer.Marshal(buf, ev)
	return buf
}

func MarshalOutgoing(buf *bytes.Buffer) *bytes.Buffer {
	msgBytes := buf.Bytes()
	keys, _ := user.NewRandKeys()
	signature := ed25519.Sign(*keys.EdSecretKey, msgBytes)

	if err := msg.VerifyEd25519Signature(keys.EdPubKeyBytes(), msgBytes, signature); err != nil {
		panic(err)
	}

	b := make([]byte, 0)
	outgoing := bytes.NewBuffer(b)
	// outgoing, _ = wire.AddTopic(outgoing, topics.Agreement)
	_ = encoding.Write512(outgoing, signature)
	_ = encoding.Write256(outgoing, keys.EdPubKeyBytes())
	if _, err := outgoing.Write(msgBytes); err != nil {
		panic(err)
	}
	return outgoing
}

func mockCommittee(quorum int, isMember bool) *mocks.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("ReportAbsentees", mock.Anything,
		mock.Anything, mock.Anything).Return(nil)
	committeeMock.On("IsMember",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(isMember)
	committeeMock.On("AmMember",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(true)
	return committeeMock
}
