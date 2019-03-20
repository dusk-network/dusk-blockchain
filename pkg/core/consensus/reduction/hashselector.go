package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type hashSelector struct {
	incomingReductionChannel chan *Event
	hashChannel              chan []byte
	resultChannel            chan []byte

	voteSet []*msg.Vote
	quorum  int
	timeOut time.Duration
}

func newHashSelector(incomingReductionChannel chan *Event,
	hashChannel, resultChannel chan []byte, quorum int,
	timeOut time.Duration) *hashSelector {

	return &hashSelector{
		incomingReductionChannel: incomingReductionChannel,
		hashChannel:              hashChannel,
		resultChannel:            resultChannel,
		quorum:                   quorum,
		timeOut:                  timeOut,
	}
}

func (hs hashSelector) sendHash(hash []byte) {
	hs.hashChannel <- hash
}

func (hs hashSelector) sendEndResult(endResult []byte) {
	hs.resultChannel <- endResult
}

// runReduction will run a two-step reduction cycle. After two steps of voting,
// the results are put into a function that deals with the outcome.
func (hs *hashSelector) runReduction() {

	// step 1
	hash1 := hs.decideOnHash()

	hs.sendHash(hash1)

	// step 2
	hash2 := hs.decideOnHash()

	if hs.reductionSuccessful(hash1, hash2) {
		result, err := hs.combineHashAndVoteSet(hash2)
		if err != nil {
			// Log
			return
		}

		hs.sendEndResult(result)
	} else {
		hs.sendEndResult(nil)
	}
}

// decideOnHash is a phase-agnostic vote tallying function. It will
// receive reduction messages through the reductionChannel for a set amount of time.
// The incoming messages will be stored in the selector's voteSet, and if one
// particular hash receives enough votes, the function returns that hash.
// If instead, the timer runs out before quorum is reached on a hash, the function
// returns the fallback value (32 empty bytes), with an empty vote set.
func (hs *hashSelector) decideOnHash() []byte {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	timer := time.NewTimer(hs.timeOut)

	for {
		select {
		case <-timer.C:
			// Return empty hash and empty vote set if we did not get a winner
			// before timeout
			return make([]byte, 32)
		case m := <-hs.incomingReductionChannel:
			votedHashStr := hex.EncodeToString(m.VotedHash)

			// Add vote for this block
			votesPerHash[votedHashStr]++

			// Add vote to vote set
			vote := createVote(m)
			hs.voteSet = append(hs.voteSet, vote)

			if int(votesPerHash[votedHashStr]) >= hs.quorum {
				// Clean up the vote set, to remove votes for other blocks.
				hs.removeDeviatingVotes(m.VotedHash)

				return m.VotedHash
			}
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

func (hs *hashSelector) removeDeviatingVotes(hash []byte) {
	for i, vote := range hs.voteSet {
		if !bytes.Equal(vote.VotedHash, hash) {
			hs.voteSet = append(hs.voteSet[:i], hs.voteSet[i+1:]...)
		}
	}
}

// combineHashAndVoteSet will return the bytes that should be sent back to the
// collector after a full reduction cycle.
func (hs hashSelector) combineHashAndVoteSet(hash []byte) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.Write256(buffer, hash); err != nil {
		return nil, err
	}

	encodedVoteSet, err := msg.EncodeVoteSet(hs.voteSet)
	if err != nil {
		return nil, err
	}

	if _, err := buffer.Write(encodedVoteSet); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// check whether a reduction cycle had a successful outcome.
func (hs hashSelector) reductionSuccessful(hash1, hash2 []byte) bool {
	notEmpty := !bytes.Equal(hash1, make([]byte, 32)) &&
		!bytes.Equal(hash2, make([]byte, 32))

	sameResults := bytes.Equal(hash1, hash2)

	voteSetsAreCorrectLength := len(hs.voteSet) >= hs.quorum*2

	return notEmpty && sameResults && voteSetsAreCorrectLength
}
