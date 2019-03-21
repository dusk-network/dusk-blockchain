package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type reductionSequencer struct {
	*reductionEventCollector
	committee    committee.Committee
	currentRound uint64
	currentStep  uint8

	timeOut                  time.Duration
	queue                    *consensus.EventQueue
	outgoingReductionChannel chan []byte
	outgoingAgreementChannel chan []byte
	voteSetStore             []*msg.Vote

	incomingReductionChannel chan *Event
	sequencerDone            bool
}

func newReductionSequencer(committee committee.Committee,
	timeOut time.Duration) *reductionSequencer {

	queue := consensus.NewEventQueue()

	reductionSequencer := &reductionSequencer{
		reductionEventCollector:  &reductionEventCollector{},
		committee:                committee,
		timeOut:                  timeOut,
		queue:                    &queue,
		outgoingReductionChannel: make(chan []byte, 1),
		outgoingAgreementChannel: make(chan []byte, 1),
	}

	return reductionSequencer
}

func (rs *reductionSequencer) process(m *Event) {
	if rs.shouldBeProcessed(m) {
		rs.incomingReductionChannel <- m
		pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
		rs.Store(m, pubKeyStr)
	} else if rs.shouldBeStored(m) {
		rs.queue.PutEvent(m.Round, m.Step, m)
	}
}

func (rs reductionSequencer) processQueuedMessages() {
	queuedEvents := rs.queue.GetEvents(rs.currentRound, rs.currentStep)

	if queuedEvents != nil {
		for _, event := range queuedEvents {
			ev := event.(*Event)
			rs.process(ev)
		}
	}
}

func (rs *reductionSequencer) incrementStep() {
	rs.currentStep++
	rs.Clear()
}

func (rs *reductionSequencer) updateRound(round uint64) {
	rs.queue.Clear(rs.currentRound)
	rs.currentRound = round
	rs.currentStep = 1
	rs.incomingReductionChannel = nil
	rs.sequencerDone = false

	rs.Clear()
}

func (rs *reductionSequencer) startSequencer(hash []byte) {
	// step 1

	// vote for the received hash
	rs.dispatchReductionVote(hash)
	// select the hash with the most votes
	hash = rs.next()

	// step 2
	rs.incrementStep()
	// vote for the returned hash
	rs.dispatchReductionVote(hash)
	// select best voted hash again
	hash = rs.next()

	hashAndVoteSet, err := rs.combineHashAndVoteSet(hash)
	if err != nil {
		// Log
		return
	}

	rs.dispatchAgreementVote(hashAndVoteSet)
	rs.incrementStep()
	// sequencer done
}

func (rs *reductionSequencer) next() []byte {
	rs.incomingReductionChannel = make(chan *Event, 100)
	rs.processQueuedMessages()
	hash, voteSet := selectHash(rs.incomingReductionChannel, rs.committee.Quorum(),
		rs.timeOut)

	rs.voteSetStore = append(rs.voteSetStore, voteSet...)
	rs.incomingReductionChannel = nil
	return hash
}

func selectHash(incomingReductionChannel chan *Event,
	quorum int, timeOut time.Duration) ([]byte, []*msg.Vote) {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	// Initalize a vote set, to add incoming votes to
	var voteSet []*msg.Vote

	timer := time.NewTimer(timeOut)

	for {
		select {
		case m := <-incomingReductionChannel:
			votedHashStr := hex.EncodeToString(m.VotedHash)

			// Add vote for this block
			votesPerHash[votedHashStr]++

			// Add vote to vote set
			vote := createVote(m)
			voteSet = append(voteSet, vote)

			if int(votesPerHash[votedHashStr]) >= quorum {
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

func (rs *reductionSequencer) combineHashAndVoteSet(hash []byte) ([]byte, error) {
	voteSetBytes, err := msg.EncodeVoteSet(rs.voteSetStore)
	if err != nil {
		return nil, err
	}

	rs.voteSetStore = nil
	return append(hash, voteSetBytes...), nil
}

func (rs reductionSequencer) shouldBeProcessed(m *Event) bool {
	pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
	isDupe := rs.Contains(pubKeyStr)
	isMember := rs.committee.IsMember(m.PubKeyBLS)
	isRelevant := rs.currentRound == m.Round && rs.currentStep == m.Step

	return !isDupe && isMember && isRelevant && rs.incomingReductionChannel != nil
}

func (rs reductionSequencer) shouldBeStored(m *Event) bool {
	futureRound := rs.currentRound <= m.Round
	futureStep := rs.currentStep <= m.Step

	return futureRound || futureStep
}

func (rs reductionSequencer) dispatchReductionVote(data []byte) {
	rs.outgoingReductionChannel <- data
}

func (rs reductionSequencer) dispatchAgreementVote(data []byte) {
	rs.outgoingAgreementChannel <- data
}
