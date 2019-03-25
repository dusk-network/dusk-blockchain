package reduction

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type eventStopWatch struct {
	collectedVotesChan chan []wire.Event
	stopChan           chan interface{}
	timer              *timer
}

func newEventStopWatch(collectedVotesChan chan []wire.Event, timer *timer) *eventStopWatch {
	return &eventStopWatch{
		collectedVotesChan: collectedVotesChan,
		stopChan:           make(chan interface{}, 1),
		timer:              timer,
	}
}

func (esw *eventStopWatch) fetch() []wire.Event {
	timer := time.NewTimer(esw.timer.timeout)
	select {
	case <-timer.C:
		esw.timer.timeoutChan <- true
		return nil
	case collectedVotes := <-esw.collectedVotesChan:
		timer.Stop()
		return collectedVotes
	case <-esw.stopChan:
		timer.Stop()
		esw.collectedVotesChan = nil
		return nil
	}
}

func (esw *eventStopWatch) stop() {
	esw.stopChan <- true
}

type reducer struct {
	firstStep  *eventStopWatch
	secondStep *eventStopWatch
	ctx        *context
}

func newCoordinator(collectedVotesChan chan []wire.Event, ctx *context) *reducer {
	return &reducer{
		firstStep:  newEventStopWatch(collectedVotesChan, ctx.timer),
		secondStep: newEventStopWatch(collectedVotesChan, ctx.timer),
		ctx:        ctx,
	}
}

func (c *reducer) begin() {
	// this is a blocking call
	events := c.firstStep.fetch()
	c.ctx.state.Step++
	hash1 := c.encodeEv(events)
	if err := c.ctx.handler.MarshalHeader(hash1, c.ctx.state); err != nil {
		panic(err)
	}
	c.ctx.reductionVoteChan <- hash1

	eventsSecondStep := c.secondStep.fetch()
	hash2 := c.encodeEv(eventsSecondStep)

	if c.isReductionSuccessful(hash1, hash2, events) {
		if err := c.ctx.handler.MarshalVoteSet(hash2, events); err != nil {
			panic(err)
		}
		if err := c.ctx.handler.MarshalHeader(hash2, c.ctx.state); err != nil {
			panic(err)
		}
		c.ctx.agreementVoteChan <- hash2
	}
	c.ctx.state.Step++
}

func (c *reducer) encodeEv(events []wire.Event) *bytes.Buffer {
	if events == nil {
		events = []wire.Event{nil}
	}
	hash := bytes.NewBuffer(make([]byte, 32))
	if err := c.ctx.handler.EmbedVoteHash(events[0], hash); err != nil {
		panic(err)
	}
	return hash
}

func (c *reducer) isReductionSuccessful(hash1, hash2 *bytes.Buffer, events []wire.Event) bool {
	bothNotNil := hash1 != nil && hash2 != nil
	identicalResults := bytes.Equal(hash1.Bytes(), hash2.Bytes())
	voteSetCorrectLength := len(events) >= c.ctx.committee.Quorum()*2

	return bothNotNil && identicalResults && voteSetCorrectLength
}

func (c *reducer) end() {
	c.firstStep.stop()
	c.secondStep.stop()
}
