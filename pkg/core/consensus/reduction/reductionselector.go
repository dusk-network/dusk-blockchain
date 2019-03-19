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
	selectorQuitChannel      chan bool

	voteSet     []*msg.Vote
	quorum      int
	timerLength time.Duration

	sentHash bool
}

func newReductionSelector(incomingReductionChannel chan *Event,
	hashChannel, resultChannel chan []byte, selectorQuitChannel chan bool,
	quorum int, timerLength time.Duration) *reductionSelector {

	return &reductionSelector{
		incomingReductionChannel: incomingReductionChannel,
		hashChannel:              hashChannel,
		resultChannel:            resultChannel,
		selectorQuitChannel:      selectorQuitChannel,
		quorum:                   quorum,
		timerLength:              timerLength,
	}
}

// Start reduction. The selector will create two channels, one to funnel incoming
// event into, and another to receive results from. This is done to prevent issues
// in the event that the collector receives a round update in the middle of a
// reduction, which would cause the connecting channels to close, causing a panic
// in the case of a delayed write from the runReduction goroutine. The seperation
// between remotely linked channels and locally linked channels prevent this from
// occurring.
func (rs *reductionSelector) initiate() {
	inChannel := make(chan *Event, 100)
	retChannel := make(chan []byte, 1)
	go rs.runReduction(inChannel, retChannel)

	for {
		select {
		case <-rs.selectorQuitChannel:
			close(rs.hashChannel)
			close(rs.resultChannel)
			return
		case m := <-rs.incomingReductionChannel:
			inChannel <- m
		case data := <-retChannel:
			// The first time, we send it to hashChannel
			if !rs.sentHash {
				rs.hashChannel <- data
				rs.sentHash = true
				break
			}

			// The second time, we send it to resultChannel, and return, as the
			// selector is done now.
			rs.resultChannel <- data
			return
		}
	}
}

// runReduction will run a two-step reduction cycle. After two steps of voting,
// the results are put into a function that deals with the outcome.
func (rs *reductionSelector) runReduction(inChannel chan *Event,
	retChannel chan []byte) {
	hash1 := rs.decideOnHash(inChannel)

	retChannel <- hash1

	hash2 := rs.decideOnHash(inChannel)

	result, err := rs.createReductionResult(hash1, hash2)
	if err != nil {
		// Log
		return
	}

	retChannel <- result
}

// decideOnHash is a phase-agnostic reduction function. It will
// receive reduction messages through the incomingReduction for a set amount of time.
// The incoming messages will be stored in the selector's voteSet, and if one
// particular hash receives enough votes, the function returns that hash.
// If instead, the timer runs out before quorum is reached on a hash, the function
// returns the fallback value (32 empty bytes), with an empty vote set.
func (rs *reductionSelector) decideOnHash(inChannel chan *Event) []byte {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	timer := time.NewTimer(rs.timerLength)

	for {
		select {
		case <-timer.C:
			// Return empty hash and empty vote set if we did not get a winner
			// before timeout
			return make([]byte, 32)
		case m := <-inChannel:
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

// createReductionResult will return the bytes that should be sent back to the
// collector after a reduction cycle.
func (rs reductionSelector) createReductionResult(hash1,
	hash2 []byte) ([]byte, error) {

	if rs.reductionSuccessful(hash1, hash2) {
		buffer := new(bytes.Buffer)
		if err := encoding.Write256(buffer, hash2); err != nil {
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

	return nil, nil
}

// check whether a reduction cycle was successfully ran or not.
func (rs reductionSelector) reductionSuccessful(hash1, hash2 []byte) bool {
	notEmpty := !bytes.Equal(hash1, make([]byte, 32)) &&
		!bytes.Equal(hash2, make([]byte, 32))

	sameResults := bytes.Equal(hash1, hash2)

	voteSetsAreCorrectLength := len(rs.voteSet) >= rs.quorum*2

	return notEmpty && sameResults && voteSetsAreCorrectLength
}
