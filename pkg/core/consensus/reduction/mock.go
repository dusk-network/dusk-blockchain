package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-crypto/bls"
)

// MockVoteSetBuffer mocks a slice of Reduction events for two adjacent steps,
// marshals them as a vote set, and returns the buffer.
func MockVoteSetBuffer(hash []byte, round uint64, step uint8, amount int) *bytes.Buffer {
	voteSet := MockVoteSet(hash, round, step, amount)
	unmarshaller := NewUnMarshaller()
	buf := new(bytes.Buffer)
	if err := unmarshaller.MarshalVoteSet(buf, voteSet); err != nil {
		panic(err)
	}

	return buf
}

// MockVoteSet mocks a slice of Reduction events for two adjacent steps,
// and returns it.
func MockVoteSet(hash []byte, round uint64, step uint8, amount int) []wire.Event {
	if step < uint8(2) {
		panic("Need at least 2 steps to create an Agreement")
	}

	votes1 := MockVotes(hash, round, step-1, amount)
	votes2 := MockVotes(hash, round, step, amount)

	return append(votes1, votes2...)
}

// MockVotes mocks a slice of Reduction events and returns it.
func MockVotes(hash []byte, round uint64, step uint8, amount int) []wire.Event {
	var voteSet []wire.Event
	for i := 0; i < amount; i++ {
		k, _ := user.NewRandKeys()
		r := MockReduction(k, hash, round, step)
		voteSet = append(voteSet, r)
	}

	return voteSet
}

// MockReduction mocks a Reduction event and returns it.
func MockReduction(keys user.Keys, hash []byte, round uint64, step uint8) *Reduction {
	reduction := MockOutgoingReduction(hash, round, step)
	reduction.PubKeyBLS = keys.BLSPubKeyBytes

	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, reduction.Header)
	sigma, _ := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, r.Bytes())
	reduction.SignedHash = sigma.Compress()
	return reduction
}

// MockOutgoingReduction adds a specified hash, round and step to an empty Reduction
// event, and returns it.
func MockOutgoingReduction(hash []byte, round uint64, step uint8) *Reduction {
	reduction := New()
	reduction.Round = round
	reduction.Step = step
	reduction.BlockHash = hash
	return reduction
}

// MockReductionBuffer mocks a Reduction event, marshals it, and returns the resulting buffer.
func MockReductionBuffer(keys user.Keys, hash []byte, round uint64, step uint8) *bytes.Buffer {
	ev := MockReduction(keys, hash, round, step)
	marshaller := NewUnMarshaller()
	buf := new(bytes.Buffer)
	_ = marshaller.Marshal(buf, ev)
	return buf
}
