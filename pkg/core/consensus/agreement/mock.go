package agreement

import (
	"bytes"
	"crypto/rand"

	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

// MockAgreementEvent returns a mocked Agreement Event, to be used for testing purposes.
func MockAgreementEvent(hash []byte, round uint64, step uint8, keys []user.Keys) *Agreement {
	a := New()
	pk, sk, _ := bls.GenKeyPair(rand.Reader)
	a.PubKeyBLS = pk.Marshal()
	a.Round = round
	a.Step = step
	a.BlockHash = hash

	// generating reduction events (votes) and signing them
	steps := GenVotes(hash, round, step, keys)
	a.VotesPerStep = steps
	buf := new(bytes.Buffer)
	_ = MarshalVotes(buf, a.VotesPerStep)
	sig, _ := bls.Sign(sk, pk, buf.Bytes())
	a.SignedVotes = sig.Compress()
	return a
}

// MockAgreement mocks an Agreement event, and returns the marshalled representation
// of it as a `*bytes.Buffer`.
// NOTE: it does not include the topic
func MockAgreement(hash []byte, round uint64, step uint8, keys []user.Keys) *bytes.Buffer {
	buf := new(bytes.Buffer)
	ev := MockAgreementEvent(hash, round, step, keys)

	marshaller := NewUnMarshaller()
	_ = marshaller.Marshal(buf, ev)
	return buf
}

func GenVotes(hash []byte, round uint64, step uint8, keys []user.Keys) []*StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	votes := make([]*StepVotes, 2)
	for i, k := range keys {

		stepCycle := i % 2
		thisStep := step + uint8(stepCycle)
		stepVote := votes[stepCycle]
		if stepVote == nil {
			stepVote = NewStepVotes()
		}

		h := &header.Header{
			BlockHash: hash,
			Round:     round,
			Step:      thisStep,
			PubKeyBLS: k.BLSPubKeyBytes,
		}

		r := new(bytes.Buffer)
		_ = header.MarshalSignableVote(r, h)
		sigma, _ := bls.Sign(k.BLSSecretKey, k.BLSPubKey, r.Bytes())

		if err := stepVote.Add(sigma.Compress(), k.BLSPubKeyBytes, thisStep); err != nil {
			panic(err)
		}
		votes[stepCycle] = stepVote
	}

	return votes
}

// MockCommittee mocks a Foldable committee implementation, which can be used for
// testing the Agreement component.
func MockCommittee(quorum int, isMember bool, membersNr int) (*mocks.Foldable, []user.Keys) {
	keys := make([]user.Keys, membersNr)
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

	committeeMock := &mocks.Foldable{}
	committeeMock.On("Quorum", mock.Anything).Return(quorum)
	committeeMock.On("IsMember",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(isMember)
	committeeMock.On("Unpack",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
		uint8(1)).Return(mockSubCommittees[0])
	committeeMock.On("Unpack",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
		uint8(2)).Return(mockSubCommittees[1])
	committeeMock.On("Pack",
		mock.Anything,
		mock.AnythingOfType("uint64"),
		uint8(1)).Return(wholeCommittee.Bits(mockSubCommittees[0]))
	committeeMock.On("Pack",
		mock.Anything,
		mock.AnythingOfType("uint64"),
		uint8(2)).Return(wholeCommittee.Bits(mockSubCommittees[1]))
	return committeeMock, keys
}
