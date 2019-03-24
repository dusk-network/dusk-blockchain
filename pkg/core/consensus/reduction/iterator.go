package reduction

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type eventStopWatch struct {
	collectedVotesChan chan []wire.Event
	stopChan           chan interface{}
	timer              *time.Timer
}

func newEventStopWatch(collectedVotesChan chan []wire.Event) *eventStopWatch {
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
	state             *consensusState
	firstStep         *eventStopWatch
	secondStep        *eventStopWatch
	reductionVoteChan chan *bytes.Buffer
	agreementVoteChan chan *bytes.Buffer
	handler           handler
	committee         committee.Committee
}

func newCoordinator(state *consensusState, collectedVotesChan chan []wire.Event, reductionVoteChan, agreementVoteChan chan *bytes.Buffer, handler handler, committee committee.Committee) *coordinator {
	return &coordinator{
		state:             state,
		firstStep:         newEventStopWatch(collectedVotesChan),
		secondStep:        newEventStopWatch(collectedVotesChan),
		reductionVoteChan: reductionVoteChan,
		agreementVoteChan: agreementVoteChan,
		handler:           handler,
		committee:         committee,
	}
}

func (c *coordinator) begin(timeout time.Duration) error {
	var hash1, hash2 *bytes.Buffer
	// this is a blocking call
	events := c.firstStep.fetch(timeout)
	if events == nil {
		// TODO: clean the stepCollector
		events = []wire.Event{nil}
	}
	c.state.Step++
	hash1 = bytes.NewBuffer(make([]byte, 32))
	if err := c.handler.EmbedVoteHash(events[0], hash1); err != nil {
		//TODO: check the impact of the error on the overall algorithm
		return err
	}
	if err := c.handler.MarshalHeader(hash1, c.state); err != nil {
		return err
	}
	c.reductionVoteChan <- hash1

	// c.voteStore = append(c.voteStore, events...)
	c.state.Step++
	if err := c.handler.EmbedVoteHash(events[0]); err != nil {
		return err
	}

	hash2 = c.secondStep.fetch(timeout)
	if c.isReductionSuccessful(hash1, hash2, events) {
		if err := c.handler.MarshalVoteSet(hash2, events); err != nil {
			return err
		}
		if err := c.handler.MarshalHeader(hash2, c.state); err != nil {
			return err
		}
		c.agreementVoteChan <- hash2
	}
	c.state++
}

func (c *coordinator) isReductionSuccessful(hash1, hash2 *bytes.Buffer, events []wire.Events) bool {
	bothNotNil := hash1 != nil && hash2 != nil
	identicalResults := bytes.Equal(hash1.Bytes(), hash2.Bytes())
	voteSetCorrectLength := len(events) >= c.committee.Quorum()*2

	return bothNotNil && identicalResults && voteSetCorrectLength
}

func (c *coordinator) end() {
	c.firstStep.stop()
	c.secondStep.stop()
}
