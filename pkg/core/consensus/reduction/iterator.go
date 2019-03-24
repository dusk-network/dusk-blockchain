package reduction

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type eventStopWatch struct {
	collectedVotesChan chan []wire.Event
	stopChan           chan interface{}
	timer              *time.Timer
}

func newEventStopWatch(collectedVotesChan chan []wire.Event) *stepStopWatch {
	return &eventStopWatch{
		collectedVotesChan: collectedVotesChan,
		stopChan:           make(chan interface{}, 1),
	}
}

func (ssw *eventStopWatch) fetch(timeout time.Duration) []wire.Event {
	ssw.timer = time.NewTimer(timeout)
	select {
	case <-ssw.timer.C:
		return nil
	case collectedVotes := <-ssw.collectedVotesChan:
		ssw.timer.Stop()
		return collectedVotes
	case <-ssw.stopChan:
		ssw.timer.Stop()
		ssw.collectedVotesChan = nil
		return nil
	}
}

func (ssw *eventStopWatch) stop() {
	ssw.stopChan <- true
}

type coordinator struct {
	state      *consensusState
	firstStep  *eventStopWatch
	secondStep *eventStopWatch
	voteStore  []wire.Event
}

func newCoordinator(state *consensusState, collectedVotesChan, reductionVoteChan, agreementVoteChan chan []wire.Event, stepChan chan uint8, timeout time.Duration) *coordinator {
	return &coordinator{
		firstStep:         newEventStopWatch(collectedVotesChan),
		secondStep:        newEventStopWatch(collectedVotesChan),
		reductionVoteChan: reductionVoteChan,
		agreementVoteChan: agreementVoteChan,
		voteStore:         make([]wire.Event, 0, 100),
	}
}

func (c *coordinator) begin(timeout time.Duration) {
	events := c.firstStep.fetch(timeout)
	c.state.Step++

	c.secondStep.fetch(timeout)
}

func (c *coordinator) end() {
	c.firstStep.stop()
	c.secondStep.stop()
}
