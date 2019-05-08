package user

import (
	"encoding/binary"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

// VotingCommittee represents a set of provisioners with voting rights at a certain
// point in the consensus. The set is sorted by the int value of the public key in increasing order (higher last)
type VotingCommittee struct {
	sortedset.Set
}

func newCommittee() *VotingCommittee {
	// return make([]*big.Int, 0)
	return &VotingCommittee{
		Set: sortedset.New(),
	}
}

func (v *VotingCommittee) Size() int {
	return len(v.Set)
}

func (v *VotingCommittee) Remove(pk []byte) bool {
	return v.Set.Remove(pk)
}

func (v *VotingCommittee) MemberKeys() [][]byte {
	pks := make([][]byte, 0)
	for _, pk := range v.Set {
		pks = append(pks, pk.Bytes())
	}
	return pks
}

func (v *VotingCommittee) Equal(other *VotingCommittee) bool {
	return v.Set.Equal(other.Set)
}

// IsMember checks if `pubKeyBLS` is within the VotingCommittee.
func (v *VotingCommittee) IsMember(pubKeyBLS []byte) bool {
	_, found := v.IndexOf(pubKeyBLS)
	return found
}

// VotingCommitteeSize returns how big the voting committee should be.
func (p *Provisioners) VotingCommitteeSize() int {
	size := p.Size()
	if size > 50 {
		size = 50
	}

	return size
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
func (p *Provisioners) CreateVotingCommittee(round, totalWeight uint64,
	step uint8) *VotingCommittee {

	votingCommittee := newCommittee()
	W := new(big.Int).SetUint64(totalWeight)
	size := p.VotingCommitteeSize()

	for i := 0; votingCommittee.Size() < size; i++ {
		hash, err := createSortitionHash(round, step, i)
		if err != nil {
			panic(err)
		}

		score := generateSortitionScore(hash, W)
		blsPk := p.extractCommitteeMember(score)
		votingCommittee.Insert(blsPk.Marshal())
	}

	return votingCommittee
}

// extractCommitteeMember walks through the committee set, while deducting
// each node's stake from the passed score until we reach zero. The public key
// of the node that the function ends on will be returned as a hexadecimal string.
func (p *Provisioners) extractCommitteeMember(score uint64) bls.PublicKey {
	for i := 0; ; i++ {
		// make sure we wrap around the provisioners array
		if i == p.Size() {
			i = 0
		}

		m := p.MemberAt(i)
		if m.Stake >= score {
			return m.PublicKeyBLS
		}

		score -= m.Stake
	}
}
