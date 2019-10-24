package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
)

// MockReduction mocks a Reduction event and returns it.
// TODO: rename it into a MockReductionEvent
func MockReduction(keys key.ConsensusKeys, hash []byte, round uint64, step uint8) Reduction {
	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: keys.BLSPubKeyBytes}
	red := New()

	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, hdr, nil)
	sigma, _ := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, r.Bytes())
	red.SignedHash = sigma.Compress()
	return *red
}

// MockReductionBuffer mocks a Reduction event, marshals it, and returns the resulting buffer.
// TODO: rename into MockWire
func MockReductionBuffer(keys key.ConsensusKeys, hash []byte, round uint64, step uint8) *bytes.Buffer {
	ev := MockReduction(keys, hash, round, step)
	buf := new(bytes.Buffer)
	_ = Marshal(buf, ev)
	return buf
}

// MockVoteSetBuffer mocks a slice of Reduction events for two adjacent steps,
// marshals them as a vote set, and returns the buffer.
func MockVoteSetBuffer(hash []byte, round uint64, step uint8, amount int) *bytes.Buffer {
	voteSet := MockVoteSet(hash, round, step, amount)
	buf := new(bytes.Buffer)
	if err := MarshalVoteSet(buf, voteSet); err != nil {
		panic(err)
	}

	return buf
}

// MockVoteSet mocks a slice of Reduction events for two adjacent steps,
// and returns it.
func MockVoteSet(hash []byte, round uint64, step uint8, amount int) []Reduction {
	if step < uint8(2) {
		panic("Need at least 2 steps to create an Agreement")
	}

	votes1 := MockVotes(hash, round, step-1, amount)
	votes2 := MockVotes(hash, round, step, amount)

	return append(votes1, votes2...)
}

// MockVotes mocks a slice of Reduction events and returns it.
func MockVotes(hash []byte, round uint64, step uint8, amount int) []Reduction {
	var voteSet []Reduction
	for i := 0; i < amount; i++ {
		k, _ := key.NewRandConsensusKeys()
		r := MockReduction(k, hash, round, step)
		voteSet = append(voteSet, r)
	}

	return voteSet
}
