package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type reductionCollector struct {
	*reductionEventCollector
	committee    user.Committee
	currentRound uint64
	currentStep  uint8

	timerLength time.Duration

	votingCommittee map[string]uint8
	queue           *reductionQueue
	inputChannel    chan *reductionEvent
	hashChannel     chan<- *bytes.Buffer
	resultChannel   chan<- *bytes.Buffer
	validate        func(*bytes.Buffer) error
	voted           bool
}

func newReductionCollector(committee user.Committee, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error,
	hashChannel, resultChannel chan *bytes.Buffer) *reductionCollector {

	queue := newReductionQueue()

	reductionCollector := &reductionCollector{
		reductionEventCollector: &reductionEventCollector{},
		committee:               committee,
		timerLength:             timerLength,
		queue:                   &queue,
		inputChannel:            make(chan *reductionEvent, 100),
		hashChannel:             hashChannel,
		resultChannel:           resultChannel,
		validate:                validateFunc,
	}

	return reductionCollector
}

func blsVerified(m *reductionEvent) bool {
	return msg.VerifyBLSSignature(m.PubKeyBLS, m.VotedHash, m.SignedHash) == nil
}

func (rc reductionCollector) shouldBeProcessed(m *reductionEvent) bool {
	hashStr := hex.EncodeToString(m.VotedHash)
	isDupe := rc.Contains(m, hashStr)

	pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
	isVoter := rc.votingCommittee[pubKeyStr] > 0

	isRelevant := rc.currentRound == m.Round && rc.currentStep == m.Step

	return !isDupe && isVoter && isRelevant
}

func (rc reductionCollector) shouldBeStored(m *reductionEvent) bool {
	futureRound := rc.currentRound < m.Round
	futureStep := rc.currentStep < m.Step

	return futureRound || futureStep
}

func (rc *reductionCollector) incrementStep() error {
	rc.currentStep++
	rc.voted = false
	return rc.setVotingCommittee()
}

func (rc *reductionCollector) updateRound(round uint64) error {
	rc.queue.Clear(rc.currentRound)
	rc.currentRound = round
	rc.currentStep = 1
	return rc.setVotingCommittee()
}

func (rc *reductionCollector) setVotingCommittee() error {
	committee, err := rc.committee.GetVotingCommittee(rc.currentRound, rc.currentStep)
	if err != nil {
		return err
	}

	rc.votingCommittee = committee
	return nil
}

func (rc reductionCollector) voteOn(hash []byte) error {
	buffer := new(bytes.Buffer)

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, rc.currentRound); err != nil {
		return err
	}

	if err := encoding.WriteUint8(buffer, rc.currentStep); err != nil {
		return err
	}

	if err := encoding.Write256(buffer, hash); err != nil {
		return err
	}

	rc.hashChannel <- buffer
	rc.voted = true
	return nil
}

// runReduction will run a two-step reduction cycle. After two steps of voting,
// the results are put into a function that deals with the outcome.
func (rc *reductionCollector) runReduction() {
	hash1, voteSet1 := rc.decideOnHash()
	if err := rc.incrementStep(); err != nil {
		// Log
		return
	}

	// Vote on the result of first step
	if err := rc.voteOn(hash1); err != nil {
		// Log
		return
	}

	hash2, voteSet2 := rc.decideOnHash()

	if err := rc.handleReductionResults(hash1, hash2, voteSet1, voteSet2); err != nil {
		// Log
		return
	}

	if err := rc.incrementStep(); err != nil {
		// Log
		return
	}
}

// decideOnHash is a phase-agnostic reduction function. It will
// receive reduction messages through the inputChannel for a set amount of time.
// The incoming messages will be stored in a VoteSet, and if one particular hash
// receives enough votes, the function returns that hash, along with the vote set.
// If instead, the timer runs out before a conclusion is reached, the function
// returns the fallback value (32 empty bytes), with an empty vote set.
func (rc *reductionCollector) decideOnHash() ([]byte, []*msg.Vote) {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	// Create a vote set variable to store incoming votes in
	var voteSet []*msg.Vote

	timer := time.NewTimer(rc.timerLength)

	for {
		select {
		case <-timer.C:
			// Return empty hash and empty vote set if we did not get a winner
			// before timeout
			return make([]byte, 32), nil
		case m := <-rc.inputChannel:
			votedHashStr := hex.EncodeToString(m.VotedHash)

			// Add votes for this block
			pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
			voteCount := rc.votingCommittee[pubKeyStr]
			votesPerHash[votedHashStr] += voteCount

			// Add vote to vote set
			vote := createVote(m)
			voteSet = append(voteSet, vote)

			// Store, so we don't count it again
			votes := rc.Store(m, votedHashStr)

			if votes >= rc.committee.Quorum() {
				rc.Clear()

				// Clean up the vote set, to remove votes for other blocks.
				removeDeviatingVotes(voteSet, m.VotedHash)

				return m.VotedHash, voteSet
			}
		}
	}
}

func createVote(m *reductionEvent) *msg.Vote {
	return &msg.Vote{
		VotedHash:  m.VotedHash,
		PubKeyBLS:  m.PubKeyBLS,
		SignedHash: m.SignedHash,
		Step:       m.Step,
	}
}

func removeDeviatingVotes(voteSet []*msg.Vote, hash []byte) {
	for i, vote := range voteSet {
		if !bytes.Equal(vote.VotedHash, hash) {
			voteSet = append(voteSet[:i], voteSet[i+1:]...)
		}
	}
}

// handleReductionResults will send the proper result to the output channel,
// depending on the passed parameters.
func (rc reductionCollector) handleReductionResults(hash1, hash2 []byte, voteSet1,
	voteSet2 []*msg.Vote) error {

	if rc.reductionSuccessful(hash1, hash2, voteSet1, voteSet2) {
		buffer := new(bytes.Buffer)
		if err := encoding.WriteUint64(buffer, binary.LittleEndian, rc.currentRound); err != nil {
			return err
		}

		if err := encoding.WriteUint8(buffer, rc.currentStep); err != nil {
			return err
		}

		if err := encoding.Write256(buffer, hash2); err != nil {
			return err
		}

		fullVoteSet := append(voteSet1, voteSet2...)
		encodedVoteSet, err := msg.EncodeVoteSet(fullVoteSet)
		if err != nil {
			return err
		}

		if _, err := buffer.Write(encodedVoteSet); err != nil {
			return err
		}

		rc.resultChannel <- buffer
	}

	return nil
}

func (rc reductionCollector) reductionSuccessful(hash1, hash2 []byte, voteSet1,
	voteSet2 []*msg.Vote) bool {

	notEmpty := !bytes.Equal(hash1, make([]byte, 32)) &&
		!bytes.Equal(hash2, make([]byte, 32))

	sameResults := bytes.Equal(hash1, hash2)

	voteSetsAreCorrectLength := len(voteSet1) >= rc.committee.Quorum() &&
		len(voteSet2) >= rc.committee.Quorum()

	return notEmpty && sameResults && voteSetsAreCorrectLength
}
