package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type reductionSelector struct {
	incomingReductionChannel chan *Event
	hashChannel              chan []byte
	resultChannel            chan []byte

	voteSet []*msg.Vote
	quorum  int
	timeOut time.Duration
}

func newReductionSelector(incomingReductionChannel chan *Event,
	hashChannel, resultChannel chan []byte, quorum int,
	timeOut time.Duration) *reductionSelector {

	return &reductionSelector{
		incomingReductionChannel: incomingReductionChannel,
		hashChannel:              hashChannel,
		resultChannel:            resultChannel,
		quorum:                   quorum,
		timeOut:                  timeOut,
	}
}

func (rs reductionSelector) sendHash(hash []byte) {
	rs.hashChannel <- hash
}

func (rs reductionSelector) sendEndResult(endResult []byte) {
	rs.resultChannel <- endResult
}

// runReduction will run a two-step reduction cycle. After two steps of voting,
// the results are put into a function that deals with the outcome.
func (rs *reductionSelector) runReduction() {

	// step 1
	hash1 := rs.decideOnHash()

	rs.sendHash(hash1)

	// step 2
	hash2 := rs.decideOnHash()

	if rs.reductionSuccessful(hash1, hash2) {
		result, err := rs.combineHashAndVoteSet(hash2)
		if err != nil {
			// Log
			return
		}

		rs.sendEndResult(result)
	} else {
		rs.sendEndResult(nil)
	}
}

// decideOnHash is a phase-agnostic vote tallying function. It will
// receive reduction messages through the reductionChannel for a set amount of time.
// The incoming messages will be stored in the selector's voteSet, and if one
// particular hash receives enough votes, the function returns that hash.
// If instead, the timer runs out before quorum is reached on a hash, the function
// returns the fallback value (32 empty bytes), with an empty vote set.
func (rs *reductionSelector) decideOnHash() []byte {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	timer := time.NewTimer(rs.timeOut)

	for {
		select {
		case <-timer.C:
			// Return empty hash and empty vote set if we did not get a winner
			// before timeout
			return make([]byte, 32)
		case m := <-rs.incomingReductionChannel:
			votedHashStr := hex.EncodeToString(m.VotedHash)

			// Add vote for this block
			votesPerHash[votedHashStr]++

			// Add vote to vote set
			vote := createVote(m)
			rs.voteSet = append(rs.voteSet, vote)

			if int(votesPerHash[votedHashStr]) >= rs.quorum {
				// Clean up the vote set, to remove votes for other blocks.
				rs.removeDeviatingVotes(m.VotedHash)

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

func (rs *reductionSelector) removeDeviatingVotes(hash []byte) {
	for i, vote := range rs.voteSet {
		if !bytes.Equal(vote.VotedHash, hash) {
			rs.voteSet = append(rs.voteSet[:i], rs.voteSet[i+1:]...)
		}
	}
}

// combineHashAndVoteSet will return the bytes that should be sent back to the
// collector after a full reduction cycle.
func (rs reductionSelector) combineHashAndVoteSet(hash []byte) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.Write256(buffer, hash); err != nil {
		return nil, err
	}

	encodedVoteSet, err := msg.EncodeVoteSet(rs.voteSet)
	if err != nil {
		return nil, err
	}

	if _, err := buffer.Write(encodedVoteSet); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// check whether a reduction cycle had a successful outcome.
func (rs reductionSelector) reductionSuccessful(hash1, hash2 []byte) bool {
	notEmpty := !bytes.Equal(hash1, make([]byte, 32)) &&
		!bytes.Equal(hash2, make([]byte, 32))

	sameResults := bytes.Equal(hash1, hash2)

	voteSetsAreCorrectLength := len(rs.voteSet) >= rs.quorum*2

	return notEmpty && sameResults && voteSetsAreCorrectLength
}
