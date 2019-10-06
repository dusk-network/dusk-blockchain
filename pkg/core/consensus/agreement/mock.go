package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
)

// MockAgreementEvent returns a mocked Agreement Event, to be used for testing purposes.
func MockAgreementEvent(hash []byte, round uint64, step uint8, keys []user.Keys, p *user.Provisioners) *Agreement {
	a := New()
	// Make sure we create an event made by an actual voting committee member
	c := p.CreateVotingCommittee(round, step, len(keys))
	cKeys := createCommitteeKeySet(c, keys)
	h := newHandler(cKeys[0])
	h.Provisioners = *p
	a.PubKeyBLS = cKeys[0].BLSPubKeyBytes
	a.Round = round
	a.Step = step
	a.BlockHash = hash

	// generating reduction events (votes) and signing them
	voteSet := CreateCommitteeVoteSet(p, keys, hash, len(keys), round, step)
	aev, err := h.Aggregate(a.Header, voteSet)
	if err != nil {
		panic(err)
	}
	buf := new(bytes.Buffer)
	_ = MarshalVotes(buf, aev.VotesPerStep)
	sig, _ := bls.Sign(cKeys[0].BLSSecretKey, cKeys[0].BLSPubKey, buf.Bytes())
	aev.SignedVotes = sig.Compress()
	return aev
}

// MockAgreement mocks an Agreement event, and returns the marshalled representation
// of it as a `*bytes.Buffer`.
// NOTE: it does not include the topic
func MockAgreement(hash []byte, round uint64, step uint8, keys []user.Keys, p *user.Provisioners) *bytes.Buffer {
	buf := new(bytes.Buffer)
	ev := MockAgreementEvent(hash, round, step, keys, p)

	marshaller := NewUnMarshaller()
	_ = marshaller.Marshal(buf, ev)
	return buf
}

// GenVotes randomly generates a slice of StepVotes with the indicated lenght.
// Albeit random, the generation is consistent with the rules of Votes
func GenVotes(hash []byte, round uint64, step uint8, keys []user.Keys, p *user.Provisioners) []*StepVotes {
	if len(keys) < 2 {
		panic("At least two votes are required to mock an Agreement")
	}

	// Create committee key sets
	keySet1 := createCommitteeKeySet(p.CreateVotingCommittee(round, step, len(keys)), keys)
	keySet2 := createCommitteeKeySet(p.CreateVotingCommittee(round, step+1, len(keys)), keys)

	stepVotes1, set1 := createStepVotesAndSet(hash, round, step, keySet1)
	stepVotes2, set2 := createStepVotesAndSet(hash, round, step+1, keySet2)

	bitSet1 := createBitSet(set1, round, step, len(keySet1), p)
	stepVotes1.BitSet = bitSet1
	bitSet2 := createBitSet(set2, round, step+1, len(keySet2), p)
	stepVotes2.BitSet = bitSet2

	return []*StepVotes{stepVotes1, stepVotes2}
}

func createStepVotesAndSet(hash []byte, round uint64, step uint8, keys []user.Keys) (*StepVotes, sortedset.Set) {
	set := sortedset.New()
	stepVotes := NewStepVotes()
	for _, k := range keys {

		// We should not aggregate any given key more than once.
		_, inserted := set.IndexOf(k.BLSPubKeyBytes)
		if !inserted {
			h := &header.Header{
				BlockHash: hash,
				Round:     round,
				Step:      step,
				PubKeyBLS: k.BLSPubKeyBytes,
			}

			r := new(bytes.Buffer)
			_ = header.MarshalSignableVote(r, h)
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

func createVoteSet(k1, k2 []user.Keys, hash []byte, size int, round uint64, step uint8) (events []wire.Event) {
	// We can not have duplicates in the vote set.
	duplicates := make(map[string]struct{})
	// We need 75% of the committee size worth of events to reach quorum.
	for j := 0; j < int(float64(size)*0.75); j++ {
		if _, ok := duplicates[string(k1[j].BLSPubKeyBytes)]; !ok {
			ev := reduction.MockReduction(k1[j], hash, round, step)
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
			ev := reduction.MockReduction(k2[j], hash, round, step+1)
			events = append(events, ev)
			duplicates[string(k2[j].BLSPubKeyBytes)] = struct{}{}
		}
	}

	return events
}

func CreateCommitteeVoteSet(p *user.Provisioners, k []user.Keys, hash []byte, committeeSize int, round uint64, step uint8) []wire.Event {
	c1 := p.CreateVotingCommittee(round, step, len(k))
	c2 := p.CreateVotingCommittee(round, step+1, len(k))
	cKeys1 := createCommitteeKeySet(c1, k)
	cKeys2 := createCommitteeKeySet(c2, k)
	events := createVoteSet(cKeys1, cKeys2, hash, len(cKeys1), round, step)
	return events
}

func createCommitteeKeySet(c user.VotingCommittee, k []user.Keys) (keys []user.Keys) {
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
