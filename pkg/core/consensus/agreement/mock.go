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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

// PublishMock is a mock-up method to facilitate testing of publishing of Agreement events
func PublishMock(bus wire.EventBroker, hash []byte, round uint64, step uint8, keys []*user.Keys) {
	buf := MockAggregatedAgreement(hash, round, step, keys)
	bus.Publish(msg.OutgoingBlockAgreementTopic, buf)
}

func MockAggregatedAgreementEvent(hash []byte, round uint64, step uint8, keys []*user.Keys) *events.AggregatedAgreement {
	if step < uint8(2) {
		panic("Need at least 2 steps to create an Agreement")
	}
	a := events.NewAggregatedAgreement()
	pk, sk, _ := bls.GenKeyPair(rand.Reader)
	a.PubKeyBLS = pk.Marshal()
	a.Round = round
	a.Step = step
	a.BlockHash = hash

	// generating reduction events (votes) and signing them
	steps := genVotes(hash, round, step, keys)
	a.VotesPerStep = steps
	buf := new(bytes.Buffer)
	_ = events.MarshalVotes(buf, a.VotesPerStep)
	sig, _ := bls.Sign(sk, pk, buf.Bytes())
	a.SignedVotes = sig.Compress()
	return a
}

func MockAggregatedAgreement(hash []byte, round uint64, step uint8, keys []*user.Keys) *bytes.Buffer {
	if step < 2 {
		panic("Aggregated agreement needs to span for at least two steps")
	}
	buf := new(bytes.Buffer)
	ev := MockAggregatedAgreementEvent(hash, round, step, keys)

	marshaller := events.NewAggregatedAgreementUnMarshaller()
	_ = marshaller.Marshal(buf, ev)
	return buf
}

func genVotes(hash []byte, round uint64, step uint8, keys []*user.Keys) []*events.StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	votes := make([]*events.StepVotes, 2)
	for i, k := range keys {
		stepCycle := i % 2
		stepVote := votes[stepCycle]
		if stepVote == nil {
			stepVote = events.NewStepVotes()
		}
		_, rEv := reduction.MockReduction(k, hash, round, step-uint8((stepCycle+1)%2))
		if err := stepVote.Add(rEv); err != nil {
			panic(err)
		}
		votes[stepCycle] = stepVote
	}
	return votes
}

func mockCommittee(quorum int, isMember bool, membersNr int) (*mocks.Committee, []*user.Keys) {
	keys := make([]*user.Keys, membersNr)
	mockSubCommittees := make([]sortedset.Set, 2)
	wholeCommittee := sortedset.New()

	// splitting the subcommittees into 2 different sets
	for i := 0; i < membersNr; i++ {
		stepCycle := i % 2
		sc := mockSubCommittees[stepCycle]
		if sc == nil {
			sc = sortedset.New()
		}
		k, _ := user.NewRandKeys()
		sc.Insert(k.BLSPubKeyBytes)
		wholeCommittee.Insert(k.BLSPubKeyBytes)
		keys[i] = k
		mockSubCommittees[stepCycle] = sc
	}

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
	committeeMock.On("Unpack",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
		uint8(1)).Return(mockSubCommittees[0])
	committeeMock.On("Unpack",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
		uint8(2)).Return(mockSubCommittees[1])
	committeeMock.On("Pack",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
		uint8(1)).Return(wholeCommittee.Bits(mockSubCommittees[0]))
	committeeMock.On("Pack",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
		uint8(2)).Return(wholeCommittee.Bits(mockSubCommittees[1]))
	return committeeMock, keys
}

// TODO: move this to reduction package
func MockAgreement(keys *user.Keys, hash []byte, round uint64, step uint8, votes []wire.Event) *events.Agreement {
	ev := events.NewAgreement()
	ev.Round = round
	ev.Step = step
	ev.PubKeyBLS = keys.BLSPubKeyBytes
	ev.VoteSet = votes
	ev.BlockHash = hash
	return ev
}

// TODO: move this to reduction package
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

// TODO: move this to reduction package
func MockAgreementBuf(hash []byte, round uint64, step uint8, voteNr int) *bytes.Buffer {
	ev := MockVoteAgreement(hash, round, step, voteNr)
	signer := voting.NewAgreementSigner(nil, nil)
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	_ = signer.Marshal(buf, ev)
	return buf
}
