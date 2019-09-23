package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
)

// MockAgreementEvent returns a mocked Agreement Event, to be used for testing purposes.
func MockAgreementEvent(hash []byte, round uint64, step uint8, keys []user.Keys, committee user.VotingCommittee) *Agreement {
	a := New()
	a.PubKeyBLS = keys[0].BLSPubKeyBytes
	a.Round = round
	a.Step = step
	a.BlockHash = hash

	// generating reduction events (votes) and signing them
	steps := GenVotes(hash, round, step, keys, committee)
	a.VotesPerStep = steps
	buf := new(bytes.Buffer)
	_ = MarshalVotes(buf, a.VotesPerStep)
	sig, _ := bls.Sign(keys[0].BLSSecretKey, keys[0].BLSPubKey, buf.Bytes())
	a.SignedVotes = sig.Compress()
	return a
}

// MockAgreement mocks an Agreement event, and returns the marshalled representation
// of it as a `*bytes.Buffer`.
// NOTE: it does not include the topic
func MockAgreement(hash []byte, round uint64, step uint8, keys []user.Keys, committee user.VotingCommittee) *bytes.Buffer {
	buf := new(bytes.Buffer)
	ev := MockAgreementEvent(hash, round, step, keys, committee)

	marshaller := NewUnMarshaller()
	_ = marshaller.Marshal(buf, ev)
	return buf
}

// GenVotes randomly generates a slice of StepVotes with the indicated lenght.
// Albeit random, the generation is consistent with the rules of Votes
func GenVotes(hash []byte, round uint64, step uint8, keys []user.Keys, committee user.VotingCommittee) []*StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	sets := make([]sortedset.Set, 2)
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
		sets[stepCycle].Insert(k.BLSPubKeyBytes)
		votes[stepCycle] = stepVote
	}

	for i, sv := range votes {
		sv.BitSet = committee.Bits(sets[i])
	}

	return votes
}
