package user

import (
	"encoding/binary"
	"errors"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// ConsensusInfo holds general information about the state of the network
// and the consensus.
type ConsensusInfo struct {
	Committee       Committee         // The global provisioner committee
	VotingCommittee map[string]uint8  // The voting committee for the current step
	NodeStakes      map[string]uint64 // Mapping of provisioner public keys to their stakes
	CommitteeSize   uint8             // Size of voting committees for the current round
	VoteLimit       uint8             // Threshold number to be reached for voting phases this round
}

// SetVotingCommittee will run the sortition function, and set the VotingCommittee value
// on ConsensusInfo.
func (c *ConsensusInfo) SetVotingCommittee(round, totalWeight uint64, step uint32) error {
	committee, err := c.CreateVotingCommittee(round, totalWeight, step)
	if err != nil {
		return err
	}

	c.VotingCommittee = committee
	return nil
}

// CreateVotingCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step.
func (c ConsensusInfo) CreateVotingCommittee(round, totalWeight uint64, step uint32) (map[string]uint8, error) {

	votingCommittee := make(map[string]uint8)
	W := new(big.Int).SetUint64(totalWeight)

	for i := uint8(0); i < c.CommitteeSize; i++ {
		// Create message to hash
		msg := createSortitionMessage(round, step, i)

		// Hash message
		hash, err := hash.Sha3256(msg)
		if err != nil {
			return nil, err
		}

		// Get a committee member
		member, err := c.getCommitteeMember(hash, W)
		if err != nil {
			return nil, err
		}

		votingCommittee[member]++
	}

	return votingCommittee, nil
}

// createSortitionMessage will create the slice of bytes containing all of the information
// we will hash during sortition, and return it.
func createSortitionMessage(round uint64, step uint32, i uint8) []byte {
	msg := make([]byte, 12)
	binary.LittleEndian.PutUint64(msg[:8], round)
	binary.LittleEndian.PutUint32(msg[8:], step)
	msg = append(msg, byte(i))

	return msg
}

// getCommitteeMember generates a score from the given hash, and then
// walks through the committee set, while deducting each node's stake from the score
// until we reach zero. The public key of the node that the function ends on
// will be returned as a hexadecimal string.
func (c ConsensusInfo) getCommitteeMember(hash []byte, W *big.Int) (string, error) {

	// Generate score
	hashNum := new(big.Int).SetBytes(hash)
	score := new(big.Int).Mod(hashNum, W).Uint64()

	// Walk through the committee set and keep deducting until we reach zero
	for _, member := range c.Committee {
		if c.NodeStakes[member.String()] >= score {
			return member.String(), nil
		}

		score -= c.NodeStakes[member.String()]
	}

	// With the ConsensusInfo Committee and TotalWeight fields properly set,
	// we should never end up here. However, if we do, return an error.
	return "", errors.New("committee and total weight values do not correspond")
}
