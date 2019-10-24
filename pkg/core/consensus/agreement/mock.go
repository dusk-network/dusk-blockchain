package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
)

// MockAgreementEvent returns a mocked Agreement Event, to be used for testing purposes.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing
func MockAgreementEvent(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, committee user.VotingCommittee, iterativeIdx ...int) *Agreement {

	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	a := New(header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: keys[idx].BLSPubKeyBytes})
	// generating reduction events (votes) and signing them
	steps := GenVotes(hash, round, step, keys, committee)

	buf := new(bytes.Buffer)
	if err := MarshalVotes(buf, steps); err != nil {
		panic(err)
	}

	whole := new(bytes.Buffer)
	if err := header.MarshalSignableVote(whole, a.Header, buf.Bytes()); err != nil {
		panic(err)
	}

	sig, _ := bls.Sign(keys[idx].BLSSecretKey, keys[idx].BLSPubKey, whole.Bytes())
	a.VotesPerStep = steps
	a.SetSignature(sig.Compress())
	return a
}

// MockWire creates a buffer representing an Agreement travelling to other Provisioners
func MockWire(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, committee user.VotingCommittee, i ...int) *bytes.Buffer {
	ev := MockAgreementEvent(hash, round, step, keys, committee, i...)

	buf := new(bytes.Buffer)
	if err := header.Marshal(buf, ev.Header); err != nil {
		panic(err)
	}

	if err := Marshal(buf, *ev); err != nil {
		panic(err)
	}
	return buf
}

// MockAgreement mocks an Agreement event, and returns the marshalled representation
// of it as a `*bytes.Buffer`.
// The `i` parameter is used to diversify the mocks to avoid duplicates
// NOTE: it does not include the topic nor the Header
func MockAgreement(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, committee user.VotingCommittee, i ...int) *bytes.Buffer {
	buf := new(bytes.Buffer)
	ev := MockAgreementEvent(hash, round, step, keys, committee, i...)
	_ = Marshal(buf, *ev)
	return buf
}

// MockConsensusEvent mocks a consensus.Event with an Agreement payload.
func MockConsensusEvent(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, committee user.VotingCommittee, i ...int) consensus.Event {
	aev := MockAgreementEvent(hash, round, step, keys, committee, i...)
	hdr := aev.Header

	buf := new(bytes.Buffer)
	_ = Marshal(buf, *aev)

	return consensus.Event{
		Header:  hdr,
		Payload: *buf,
	}
}

// GenVotes randomly generates a slice of StepVotes with the indicated lenght.
// Albeit random, the generation is consistent with the rules of Votes
func GenVotes(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, committee user.VotingCommittee) []*StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	votes := make([]*StepVotes, 2)
	sets := make([]sortedset.Set, 2)

	for i, k := range keys {

		stepCycle := i % 2
		thisStep := step + uint8(stepCycle)
		stepVote := votes[stepCycle]
		if stepVote == nil {
			stepVote = NewStepVotes()
		}

		h := header.Header{
			BlockHash: hash,
			Round:     round,
			Step:      thisStep,
			PubKeyBLS: k.BLSPubKeyBytes,
		}

		r := new(bytes.Buffer)
		_ = header.MarshalSignableVote(r, h, nil)
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
