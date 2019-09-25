package user

import (
	"encoding/binary"
	"math"
	"math/big"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/hash"
)

// VotingCommittee represents a set of provisioners with voting rights at a certain
// point in the consensus. The set is sorted by the int value of the public key in
// increasing order (higher last)
type VotingCommittee struct {
	sortedset.Set
}

func newCommittee() *VotingCommittee {
	return &VotingCommittee{
		Set: sortedset.New(),
	}
}

// Size returns how many members there are in a VotingCommittee.
func (v VotingCommittee) Size() int {
	return len(v.Set)
}

// Remove a member from the VotingCommittee by its BLS public key.
func (v *VotingCommittee) Remove(pk []byte) bool {
	return v.Set.Remove(pk)
}

// MemberKeys returns the BLS public keys of all the members in a VotingCommittee.
func (v VotingCommittee) MemberKeys() [][]byte {
	pks := make([][]byte, 0)
	for _, pk := range v.Set {
		pks = append(pks, pk.Bytes())
	}
	return pks
}

// Equal checks if two VotingCommittees are the same.
func (v VotingCommittee) Equal(other *VotingCommittee) bool {
	return v.Set.Equal(other.Set)
}

// IsMember checks if `pubKeyBLS` is within the VotingCommittee.
func (v VotingCommittee) IsMember(pubKeyBLS []byte) bool {
	_, found := v.IndexOf(pubKeyBLS)
	return found
}

// createSortitionMessage will return the hash of the passed sortition information.
func createSortitionHash(round uint64, step uint8, i int) ([]byte, error) {
	msg := make([]byte, 12)
	binary.LittleEndian.PutUint64(msg[:8], round)
	binary.LittleEndian.PutUint32(msg[8:12], uint32(i))
	msg = append(msg, byte(step))
	return hash.Sha3256(msg)
}

// generate a score from the given hash and total stake weight
func generateSortitionScore(hash []byte, W *big.Int) uint64 {
	hashNum := new(big.Int).SetBytes(hash)
	return new(big.Int).Mod(hashNum, W).Uint64()
}

// CreateVotingCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step and round.
func (p Provisioners) CreateVotingCommittee(round uint64, step uint8, size int) VotingCommittee {
	votingCommittee := newCommittee()
	W := new(big.Int).SetUint64(p.TotalWeight())

	for i := 0; votingCommittee.Size() < size; i++ {
		hash, err := createSortitionHash(round, step, i)
		if err != nil {
			panic(err)
		}

		score := generateSortitionScore(hash, W)
		idx, blsPk := p.extractCommitteeMember(score)
		if !votingCommittee.Insert(blsPk) {
			for {
				idx++
				if idx >= len(p.Members) {
					idx = 0
				}
				m := p.MemberAt(idx)
				if votingCommittee.Insert(m.PublicKeyBLS) {
					break
				}
			}
		}
	}

	return *votingCommittee
}

// extractCommitteeMember walks through the committee set, while deducting
// each node's stake from the passed score until we reach zero. The public key
// of the node that the function ends on will be returned as a hexadecimal string.
func (p Provisioners) extractCommitteeMember(score uint64) (int, []byte) {
	for i := 0; ; i++ {
		// make sure we wrap around the provisioners array
		if i >= len(p.Members) {
			i = 0
		}

		m := p.MemberAt(i)
		stake, err := p.GetStake(m.PublicKeyBLS)
		if err != nil {
			// If we get an error from GetStake, it means we either got a public key of a
			// provisioner who is no longer in the set, or we got a malformed public key.
			// We can't repair our committee on the fly, so we have to panic.
			panic(err)
		}

		if stake >= score {
			return i, m.PublicKeyBLS
		}

		score -= stake
	}
}

func (p Provisioners) GenerateCommittees(round uint64, amount, step uint8, size int) []VotingCommittee {
	if step >= math.MaxUint8-amount {
		amount = math.MaxUint8 - step
	}

	committees := make([]VotingCommittee, amount)
	for i := 0; i < int(amount); i++ {
		votingCommittee := p.CreateVotingCommittee(round, step+uint8(i), size)
		committees[i] = votingCommittee
	}

	return committees
}
