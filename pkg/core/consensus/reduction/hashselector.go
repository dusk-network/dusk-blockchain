package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type hashSelector struct {
	incomingReductionChannel chan *Event
	stopChannel              chan bool
	quorum                   int
	timeOut                  time.Duration
}

func newHashSelector(quorum int, timeOut time.Duration) *hashSelector {
	return &hashSelector{
		incomingReductionChannel: make(chan *Event, 100),
		stopChannel:              make(chan bool, 1),
		quorum:                   quorum,
		timeOut:                  timeOut,
	}
}

// decideOnHash is a phase-agnostic vote tallying function. It will
// receive reduction messages through the reductionChannel for a set amount of time.
// The incoming messages will be stored in the selector's voteSet, and if one
// particular hash receives enough votes, the function returns that hash.
// If instead, the timer runs out before quorum is reached on a hash, the function
// returns the fallback value (32 empty bytes), with an empty vote set.
func (hs *hashSelector) selectHash() ([]byte, []*msg.Vote) {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	// Initalize a vote set, to add incoming votes to
	var voteSet []*msg.Vote

	timer := time.NewTimer(hs.timeOut)

	for {
		select {
		case m := <-hs.incomingReductionChannel:
			votedHashStr := hex.EncodeToString(m.VotedHash)

			// Add vote for this block
			votesPerHash[votedHashStr]++

			// Add vote to vote set
			vote := createVote(m)
			voteSet = append(voteSet, vote)

			if int(votesPerHash[votedHashStr]) >= hs.quorum {
				// Clean up the vote set, to remove votes for other blocks.
				for i, vote := range voteSet {
					if !bytes.Equal(vote.VotedHash, m.VotedHash) {
						voteSet = append(voteSet[:i], voteSet[i+1:]...)
					}
				}

				return m.VotedHash, voteSet
			}
		case <-timer.C:
			// Return empty hash and empty vote set if we did not get a winner
			// before timeout
			return make([]byte, 32), nil
		case <-hs.stopChannel:
			return nil, nil
		}
	}
}

func createVote(m *Event) *msg.Vote {
	return &msg.Vote{
		VotedHash:  m.VotedHash,
		PubKeyBLS:  m.PubKeyBLS,
		SignedHash: m.SignedHash,
		Step:       m.Step,
	}
}
