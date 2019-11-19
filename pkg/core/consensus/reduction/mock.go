package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
)

// MockEvent mocks a Reduction event and returns it.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing
func MockEvent(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, iterativeIdx ...int) (Reduction, header.Header) {
	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: keys[idx].BLSPubKeyBytes}
	red := New()

	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, hdr)
	sigma, _ := bls.Sign(keys[idx].BLSSecretKey, keys[idx].BLSPubKey, r.Bytes())
	red.SignedHash = sigma.Compress()
	return *red, hdr
}

// MockWire mocks an outgoing Reduction event, marshals it, and returns the resulting buffer.
// Note: it does not Edward sign the message
func MockWire(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, iterativeIdx ...int) *bytes.Buffer {
	ev, hdr := MockEvent(hash, round, step, keys, iterativeIdx...)
	buf := new(bytes.Buffer)
	_ = header.Marshal(buf, hdr)
	_ = Marshal(buf, ev)
	return buf
}

// MockConsensusEvent mocks a Reduction event already digested by the Coordinator
func MockConsensusEvent(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, iterativeIdx ...int) consensus.Event {
	rev, hdr := MockEvent(hash, round, step, keys, iterativeIdx...)

	buf := new(bytes.Buffer)
	_ = Marshal(buf, rev)

	return consensus.Event{
		Header:  hdr,
		Payload: *buf,
	}
}

// MockVoteSetBuffer mocks a slice of Reduction events for two adjacent steps,
// marshals them as a vote set, and returns the buffer.
func MockVoteSetBuffer(hash []byte, round uint64, step uint8, amount int, keys []key.ConsensusKeys) *bytes.Buffer {
	voteSet := MockVoteSet(hash, round, step, keys, amount)
	buf := new(bytes.Buffer)
	if err := MarshalVoteSet(buf, voteSet); err != nil {
		panic(err)
	}

	return buf
}

// MockVoteSet mocks a slice of Reduction events for two adjacent steps,
// and returns it.
func MockVoteSet(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, amount int) []Reduction {
	if step < uint8(2) {
		panic("Need at least 2 steps to create an Agreement")
	}

	votes1 := MockVotes(hash, round, step-1, keys, amount)
	votes2 := MockVotes(hash, round, step, keys, amount)

	return append(votes1, votes2...)
}

// MockVotes mocks a slice of Reduction events and returns it.
func MockVotes(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, amount int) []Reduction {
	var voteSet []Reduction
	for i := 0; i < amount; i++ {
		r, _ := MockEvent(hash, round, step, keys, i)
		voteSet = append(voteSet, r)
	}

	return voteSet
}
