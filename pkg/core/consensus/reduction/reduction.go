package reduction

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// DecideOnHash is a phase-agnostic reduction function. It will receive reduction
// messages through the reductionChannel for a set amount of time. The incoming
// messages will be stored in a VoteSet, and if one certain hash receives
// enough votes, the function will return that hash, along with the accumulated votes.
func DecideOnHash(timerLength time.Duration, info user.ConsensusInfo,
	reductionChannel chan *payload.MsgConsensus) ([]byte, user.VoteSet, error) {

	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	// Create a VoteSet to store incoming votes in
	var voteSet user.VoteSet

	timer := time.NewTimer(timerLength)

	for {
		select {
		case <-timer.C:
			return make([]byte, 32), nil, nil
		case m := <-reductionChannel:
			// Get the amount of votes that this node is allowed this step.
			pubKeyStr := hex.EncodeToString(m.PubKey)
			votes := info.VotingCommittee[pubKeyStr]

			// Increase the vote counter for the included hash.
			pl := m.Payload.(*payload.Reduction)
			hashStr := hex.EncodeToString(pl.Hash)
			votesPerHash[hashStr] += votes

			// Add the vote to the vote set.
			if err := voteSet.AddVote(pl, m.Step, votes); err != nil {
				return nil, nil, err
			}

			// Check if we have exceeded the limit.
			if limitExceeded(votesPerHash[m.Step], info.VoteLimit) {
				// If so, we clean up the vote set, to remove votes for other blocks.
				voteSet.RemoveDeviatingVotes(pl.Hash)

				// Then, return the hash and the vote set.
				return pl.Hash, voteSet, nil
			}
		}
	}
}

func limitExceeded(votes, voteLimit uint8) bool {
	return votes >= voteLimit
}
