package reduction

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type stepStopWatch struct {
	resultChan         chan []wire.Event
	collectedVotesChan chan []wire.Event
	stopChan           chan interface{}
	timer              *time.Timer
}

func newStepStopWatch(resultChan, collectedVotesChan chan []wire.Event) *stepStopWatch {
	return &stepStopWatch{
		resultChan:         resultChan,
		collectedVotesChan: collectedVotesChan,
		stopChan:           make(chan interface{}, 1),
	}
}

func (ssw *stepStopWatch) start(timeout time.Duration) {
	ssw.timer = time.NewTimer(timeout)
	select {
	case <-ssw.timer.C:
		ssw.resultChan <- nil
	case collectedVotes := <-ssw.collectedVotesChan:
		ssw.timer.Stop()
		ssw.resultChan <- collectedVotes
	case <-ssw.stopChan:
		ssw.timer.Stop()
		ssw.collectedVotesChan = nil
	}
}

func (ssw *stepStopWatch) stop() {
	ssw.stopChan <- true
}

type coordinator struct {
	firstStep  *stepStopWatch
	secondStep *stepStopWatch
	voteStore  []wire.Event
}

func newCoordinator(collectedVotesChan, reductionVoteChan, agreementVoteChan chan []wire.Event, timeout time.Duration) *coordinator {
	return &coordinator{
		firstStep:  newStepStopWatch(reductionVoteChan, collectedVotesChan),
		secondStep: newStepStopWatch(agreementVoteChan, collectedVotesChan),
		voteStore:  make([]wire.Event, 0, 100),
	}
}

func (c *coordinator) begin(timeout time.Duration) {
	c.firstStep.start(timeout)
	c.secondStep.start(timeout)
}

func (c *coordinator) end() {
	c.firstStep.stop()
	c.secondStep.stop()
}
