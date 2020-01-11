package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
)

// MockAgreementEvent returns a mocked Agreement Event, to be used for testing purposes.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing
func MockAgreementEvent(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners, iterativeIdx ...int) *message.Agreement {
	// Make sure we create an event made by an actual voting committee member
	c := p.CreateVotingCommittee(round, step, len(keys))
	cKeys := createCommitteeKeySet(c, keys)

	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	a := message.NewAgreement(header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: cKeys[idx].BLSPubKeyBytes})
	// generating reduction events (votes) and signing them
	steps := GenVotes(hash, round, step, keys, p)

	whole := new(bytes.Buffer)
	if err := header.MarshalSignableVote(whole, a.Header); err != nil {
		panic(err)
	}

	sig, _ := bls.Sign(cKeys[idx].BLSSecretKey, cKeys[idx].BLSPubKey, whole.Bytes())
	a.VotesPerStep = steps
	a.SetSignature(sig.Compress())
	return a
}

// MockWire creates a buffer representing an Agreement travelling to other Provisioners
func MockWire(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners, i ...int) *bytes.Buffer {
	ev := MockAgreementEvent(hash, round, step, keys, p, i...)

	buf := new(bytes.Buffer)
	if err := header.Marshal(buf, ev.Header); err != nil {
		panic(err)
	}

	if err := message.MarshalAgreement(buf, *ev); err != nil {
		panic(err)
	}
	return buf
}

// MockAgreement mocks an Agreement event, and returns the marshalled representation
// of it as a `*bytes.Buffer`.
// The `i` parameter is used to diversify the mocks to avoid duplicates
// NOTE: it does not include the topic nor the Header
func MockAgreement(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners, i ...int) *bytes.Buffer {
	buf := new(bytes.Buffer)
	ev := MockAgreementEvent(hash, round, step, keys, p, i...)
	_ = message.MarshalAgreement(buf, *ev)
	return buf
}

// MockConsensusEvent mocks a consensus.Event with an Agreement payload.
func MockConsensusEvent(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners, i ...int) consensus.Event {
	aev := MockAgreementEvent(hash, round, step, keys, p, i...)
	hdr := aev.Header

	buf := new(bytes.Buffer)
	_ = message.MarshalAgreement(buf, *aev)

	return consensus.Event{
		Header:  hdr,
		Payload: *buf,
	}
}

// GenVotes randomly generates a slice of StepVotes with the indicated lenght.
// Albeit random, the generation is consistent with the rules of Votes
func GenVotes(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, p *user.Provisioners) []*message.StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	// Create committee key sets
	keySet1 := createCommitteeKeySet(p.CreateVotingCommittee(round, step-2, len(keys)), keys)
	keySet2 := createCommitteeKeySet(p.CreateVotingCommittee(round, step-1, len(keys)), keys)

	stepVotes1, set1 := createStepVotesAndSet(hash, round, step-2, keySet1)
	stepVotes2, set2 := createStepVotesAndSet(hash, round, step-1, keySet2)

	bitSet1 := createBitSet(set1, round, step-2, len(keySet1), p)
	stepVotes1.BitSet = bitSet1
	bitSet2 := createBitSet(set2, round, step-1, len(keySet2), p)
	stepVotes2.BitSet = bitSet2

	return []*message.StepVotes{stepVotes1, stepVotes2}
}

func createStepVotesAndSet(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys) (*message.StepVotes, sortedset.Set) {
	set := sortedset.New()
	stepVotes := message.NewStepVotes()
	for _, k := range keys {

		// We should not aggregate any given key more than once.
		_, inserted := set.IndexOf(k.BLSPubKeyBytes)
		if !inserted {
			h := header.Header{
				BlockHash: hash,
				Round:     round,
				Step:      step,
				PubKeyBLS: k.BLSPubKeyBytes,
			}

			r := new(bytes.Buffer)
			if err := header.MarshalSignableVote(r, h); err != nil {
				panic(err)
			}

			sigma, _ := bls.Sign(k.BLSSecretKey, k.BLSPubKey, r.Bytes())
			if err := stepVotes.Add(sigma.Compress(), k.BLSPubKeyBytes, step); err != nil {
				panic(err)
			}
		}

		set.Insert(k.BLSPubKeyBytes)
	}

	return stepVotes, set
}

func createBitSet(set sortedset.Set, round uint64, step uint8, size int, p *user.Provisioners) uint64 {
	committee := p.CreateVotingCommittee(round, step, size)
	return committee.Bits(set)
}

func CreateCommitteeVoteSet(p *user.Provisioners, k []key.ConsensusKeys, hash []byte, committeeSize int, round uint64, step uint8) []consensus.Event {
	c1 := p.CreateVotingCommittee(round, step-2, len(k))
	c2 := p.CreateVotingCommittee(round, step-1, len(k))
	cKeys1 := createCommitteeKeySet(c1, k)
	cKeys2 := createCommitteeKeySet(c2, k)
	events := createVoteSet(cKeys1, cKeys2, hash, len(cKeys1), round, step)
	return events
}

func createCommitteeKeySet(c user.VotingCommittee, k []key.ConsensusKeys) (keys []key.ConsensusKeys) {
	committeeKeys := c.MemberKeys()

	for _, cKey := range committeeKeys {
		for _, key := range k {
			if bytes.Equal(cKey, key.BLSPubKeyBytes) {
				keys = append(keys, key)
				break
			}
		}
	}

	return keys
}

// createVoteSet creates and returns a set of Reduction votes for two steps.
func createVoteSet(k1, k2 []key.ConsensusKeys, hash []byte, size int, round uint64, step uint8) (events []consensus.Event) {
	// We can not have duplicates in the vote set.
	duplicates := make(map[string]struct{})
	// We need 75% of the committee size worth of events to reach quorum.
	for j := 0; j < int(float64(size)*0.75); j++ {
		if _, ok := duplicates[string(k1[j].BLSPubKeyBytes)]; !ok {
			ev := mockReduction(hash, round, step-2, k1, j)
			events = append(events, ev)
			duplicates[string(k1[j].BLSPubKeyBytes)] = struct{}{}
		}
	}

	// Clear the duplicates map, since we will most likely have identical keys in each array
	for k := range duplicates {
		delete(duplicates, k)
	}

	for j := 0; j < int(float64(size)*0.75); j++ {
		if _, ok := duplicates[string(k2[j].BLSPubKeyBytes)]; !ok {
			ev := mockReduction(hash, round, step-1, k2, j)
			events = append(events, ev)
			duplicates[string(k2[j].BLSPubKeyBytes)] = struct{}{}
		}
	}

	return events
}

// TODO: this is incredibly similar to reduction.MockEvent, but is placed here due to
// issues with circular imports. we should find a way to remove this duplicated code
func mockReduction(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, iterativeIdx ...int) consensus.Event {
	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: keys[idx].BLSPubKeyBytes}

	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, hdr)
	sigma, _ := bls.Sign(keys[idx].BLSSecretKey, keys[idx].BLSPubKey, r.Bytes())
	signedHash := sigma.Compress()
	return consensus.Event{hdr, *bytes.NewBuffer(signedHash)}
}
