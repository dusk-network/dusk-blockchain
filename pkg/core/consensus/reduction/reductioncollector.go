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

	timerLength time.Duration

	queue *reductionQueue

	// channels linked to the selector
	incomingReductionChannel chan *Event
	selectorQuitChannel      chan bool
	hashChannel              chan []byte
	resultChannel            chan []byte

	// keep track of whether or not we have voted on the current step
	votedThisStep bool
}

func newReductionCollector(committee committee.Committee,
	timerLength time.Duration) *reductionCollector {

	queue := newReductionQueue()

	reductionCollector := &reductionCollector{
		reductionEventCollector: &reductionEventCollector{},
		committee:               committee,
		timerLength:             timerLength,
		queue:                   &queue,
	}

	return reductionCollector
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

func (rc *reductionCollector) incrementStep() {
	rc.currentStep++
	rc.votedThisStep = false
	rc.Clear()
}

func (rc reductionCollector) startSelector() {
	// create channels to communicate with the selector
	rc.incomingReductionChannel = make(chan *Event, 100)
	rc.selectorQuitChannel = make(chan bool, 1)
	rc.hashChannel = make(chan []byte, 1)
	rc.resultChannel = make(chan []byte, 1)

	// create and start the selector
	rs := newReductionSelector(rc.incomingReductionChannel, rc.hashChannel,
		rc.resultChannel, rc.selectorQuitChannel, rc.committee.Quorum(),
		rc.timerLength)

	go rs.initiate()
}

func (rc reductionCollector) stopSelector() {
	rc.selectorQuitChannel <- true
	close(rc.incomingReductionChannel)
	close(rc.selectorQuitChannel)
}
