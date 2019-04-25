package user

import (
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

var _empty struct{}

// VotingCommittee represents a set of provisioners with voting rights at a certain
// point in the consensus.
type VotingCommittee map[string]struct{}

// IsMember checks if there is a map entry for `pubKeyBLS`.
func (v VotingCommittee) IsMember(pubKeyBLS []byte) bool {
	_, ok := v[hex.EncodeToString(pubKeyBLS)]
	return ok
}

// CreateVotingCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step and round.
func (p Provisioners) CreateVotingCommittee(round, totalWeight uint64,
	step uint8) VotingCommittee {

	votingCommittee := make(map[string]struct{})
	W := new(big.Int).SetUint64(totalWeight)
	size := p.VotingCommitteeSize()

	for i := 0; len(votingCommittee) < size; i++ {
		hash, err := createSortitionHash(round, step, i)
		if err != nil {
			panic(err)
		}

		score := generateSortitionScore(hash, W)
		member := p.extractCommitteeMember(score)
		votingCommittee[member] = _empty
	}

	return votingCommittee
}

// VotingCommitteeSize returns how big the voting committee should be.
func (p Provisioners) VotingCommitteeSize() int {
	size := len(p)
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

// extractCommitteeMember walks through the committee set, while deducting
// each node's stake from the passed score until we reach zero. The public key
// of the node that the function ends on will be returned as a hexadecimal string.
func (p Provisioners) extractCommitteeMember(score uint64) string {
	for i := 0; ; i++ {
		// make sure we wrap around the provisioners array
		if i == len(p) {
			i = 0
		}

		if p[i].Stake >= score {
			return p[i].BLSString()
		}

		score -= p[i].Stake
	}
}
