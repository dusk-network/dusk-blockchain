package reduction

import (
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

	currentSelector *hashSelector
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
		rs.currentSelector.incomingReductionChannel <- m
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
	rs.currentSelector = nil
	rs.Clear()
}

func (rs *reductionSequencer) updateRound(round uint64) {
	rs.queue.Clear(rs.currentRound)
	rs.currentRound = round
	rs.currentStep = 1
	rs.currentSelector = nil

	rs.Clear()
}

func (rs *reductionSequencer) startSequencer(hash []byte) {
	// step 1
	rs.dispatchReductionVote(hash)
	hash = rs.next()

	// step 2
	rs.incrementStep()
	rs.dispatchReductionVote(hash)
	hash = rs.next()

	voteSetBytes, err := msg.EncodeVoteSet(rs.voteSetStore)
	if err != nil {
		// Log
		return
	}

	rs.voteSetStore = nil
	hashAndVoteSet := append(hash, voteSetBytes...)
	rs.dispatchAgreementVote(hashAndVoteSet)
	rs.incrementStep()
	// sequencer done
}

func (rs *reductionSequencer) next() []byte {
	// Flush queue
	rs.processQueuedMessages()

	selector := newHashSelector(rs.committee.Quorum(), rs.timeOut)
	rs.currentSelector = selector
	hash, voteSet := selector.selectHash()

	rs.voteSetStore = append(rs.voteSetStore, voteSet...)
	return hash
}

func (rs reductionSequencer) shouldBeProcessed(m *Event) bool {
	pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
	isDupe := rs.Contains(pubKeyStr)
	isMember := rs.committee.IsMember(m.PubKeyBLS)
	isRelevant := rs.currentRound == m.Round && rs.currentStep == m.Step

	return !isDupe && isMember && isRelevant && rs.currentSelector != nil
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
