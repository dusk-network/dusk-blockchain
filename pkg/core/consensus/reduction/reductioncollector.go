package reduction

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type reductionCollector struct {
	*reductionEventCollector
	committee    committee.Committee
	currentRound uint64
	currentStep  uint8

	timeOut time.Duration

	queue *reductionQueue

	// channels linked to the selector
	incomingReductionChannel chan *Event
	hashChannel              chan []byte
	resultChannel            chan []byte

	// keep track of whether or not we have voted on the current step
	votedThisStep bool

	// this lets us know if there is a selector active
	reducing bool
}

func newReductionCollector(committee committee.Committee,
	timeOut time.Duration) *reductionCollector {

	queue := newReductionQueue()

	reductionCollector := &reductionCollector{
		reductionEventCollector: &reductionEventCollector{},
		committee:               committee,
		timeOut:                 timeOut,
		queue:                   &queue,
	}

	return reductionCollector
}

func (rc *reductionCollector) process(m *Event) {
	if rc.shouldBeProcessed(m) && blsVerified(m) {
		if !rc.reducing {
			rc.reducing = true
			go rc.startSelector()
		}

		rc.incomingReductionChannel <- m
		pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
		rc.Store(m, pubKeyStr)
	} else if rc.shouldBeStored(m) && blsVerified(m) {
		rc.queue.PutMessage(m.Round, m.Step, m)
	}
}

func (rc reductionCollector) getQueuedMessages() {
	queuedMessages := rc.queue.GetMessages(rc.currentRound, rc.currentStep)

	if queuedMessages != nil {
		for _, message := range queuedMessages {
			m := message.(*BlockReduction)
			rc.process(m)
		}
	}
}

func (rc *reductionCollector) incrementStep() {
	rc.currentStep++
	rc.votedThisStep = false
	rc.Clear()
	rc.getQueuedMessages()
}

func (rc reductionCollector) startSelector() {
	// create channels to communicate with the selector
	rc.incomingReductionChannel = make(chan *Event, 100)
	rc.hashChannel = make(chan []byte, 1)
	rc.resultChannel = make(chan []byte, 1)

	// create a new selector and start it
	rs := newHashSelector(rc.incomingReductionChannel, rc.hashChannel,
		rc.resultChannel, rc.committee.Quorum(), rc.timeOut)

	go rs.runReduction()
}

func (rc reductionCollector) disconnectSelector() {
	rc.incomingReductionChannel = nil
	rc.hashChannel = nil
	rc.resultChannel = nil
}

func blsVerified(m *Event) bool {
	return msg.VerifyBLSSignature(m.PubKeyBLS, m.VotedHash, m.SignedHash) == nil
}

func (rc reductionCollector) shouldBeProcessed(m *Event) bool {
	pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
	isDupe := rc.Contains(pubKeyStr)

	isMember := rc.committee.IsMember(m.PubKeyBLS)

	isRelevant := rc.currentRound == m.Round && rc.currentStep == m.Step

	return !isDupe && isMember && isRelevant
}

func (rc reductionCollector) shouldBeStored(m *Event) bool {
	futureRound := rc.currentRound < m.Round
	futureStep := rc.currentStep < m.Step

	return futureRound || futureStep
}
