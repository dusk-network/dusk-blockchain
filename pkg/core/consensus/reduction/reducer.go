package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type eventStopWatch struct {
	collectedVotesChan chan []wire.Event
	stopChan           chan interface{}
	timer              *consensus.Timer
}

func newEventStopWatch(collectedVotesChan chan []wire.Event, timer *consensus.Timer) *eventStopWatch {
	return &eventStopWatch{
		collectedVotesChan: collectedVotesChan,
		stopChan:           make(chan interface{}, 1),
		timer:              timer,
	}
}

func (esw *eventStopWatch) fetch() []wire.Event {
	timer := time.NewTimer(esw.timer.Timeout)
	select {
	case <-timer.C:
		esw.timer.TimeoutChan <- true
		stop(timer)
		return nil
	case collectedVotes := <-esw.collectedVotesChan:
		stop(timer)
		return collectedVotes
	case <-esw.stopChan:
		stop(timer)
		esw.collectedVotesChan = nil
		return nil
	}
}

func stop(t *time.Timer) {
	if t != nil {
		t.Stop()
		t = nil
	}
}

func (esw *eventStopWatch) stop() {
	esw.stopChan <- true
}

type reducer struct {
	firstStep  *eventStopWatch
	secondStep *eventStopWatch
	ctx        *context
	stale      bool

	publisher wire.EventPublisher
}

func newReducer(collectedVotesChan chan []wire.Event, ctx *context,
	publisher wire.EventPublisher) *reducer {

	return &reducer{
		firstStep:  newEventStopWatch(collectedVotesChan, ctx.timer),
		secondStep: newEventStopWatch(collectedVotesChan, ctx.timer),
		ctx:        ctx,
		publisher:  publisher,
	}
}

func (c *reducer) requestUpdate(command func()) {
	if !c.stale {
		command()
	}
}

func (c *reducer) sendEvent(topic string, msg *bytes.Buffer) {
	if !c.stale {
		c.publisher.Publish(topic, msg)
	}
}

func (c *reducer) begin() {
	log.WithField("process", "reducer").Traceln("Beginning Reduction")
	// this is a blocking call
	events := c.firstStep.fetch()
	log.WithField("process", "reducer").Traceln("First step completed")
	c.requestUpdate(c.ctx.state.IncrementStep)
	hash1 := c.encodeEv(events)
	reductionVote, err := c.ctx.handler.MarshalHeader(hash1, c.ctx.state)
	if err != nil {
		panic(err)
	}
	c.sendEvent(string(msg.OutgoingBlockReductionTopic), reductionVote)
	eventsSecondStep := c.secondStep.fetch()
	log.WithField("process", "reducer").Traceln("Second step completed")
	hash2 := c.encodeEv(eventsSecondStep)

	allEvents := append(events, eventsSecondStep...)

	if c.isReductionSuccessful(hash1, hash2, allEvents) {

		log.WithFields(log.Fields{
			"process":    "reducer",
			"votes":      len(allEvents),
			"block hash": hex.EncodeToString(hash1.Bytes()),
		}).Debugln("Reduction successful")
		if err := c.ctx.handler.MarshalVoteSet(hash2, allEvents); err != nil {
			panic(err)
		}
		agreementVote, err := c.ctx.handler.MarshalHeader(hash2, c.ctx.state)
		if err != nil {
			panic(err)
		}

		c.sendEvent(string(msg.OutgoingBlockAgreementTopic), agreementVote)
	}

	c.requestUpdate(func() {
		c.ctx.state.IncrementStep()
		log.WithFields(log.Fields{
			"process": "reduction",
			"round":   c.ctx.state.Round(),
			"step":    c.ctx.state.Step(),
		}).Debug("Reduction complete")
		roundAndStep := make([]byte, 8)
		binary.LittleEndian.PutUint64(roundAndStep, c.ctx.state.Round())
		roundAndStep = append(roundAndStep, byte(c.ctx.state.Step()))
		c.sendEvent(string(msg.BlockGenerationTopic), bytes.NewBuffer(roundAndStep))
	})
}

func (c *reducer) encodeEv(events []wire.Event) *bytes.Buffer {
	if events == nil {
		events = []wire.Event{nil}
	}
	hash := new(bytes.Buffer)
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
