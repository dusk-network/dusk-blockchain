package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var empty struct{}

type eventStopWatch struct {
	collectedVotesChan chan []wire.Event
	stopChan           chan struct{}
	timer              *consensus.Timer
}

func newEventStopWatch(collectedVotesChan chan []wire.Event, timer *consensus.Timer) *eventStopWatch {
	return &eventStopWatch{
		collectedVotesChan: collectedVotesChan,
		stopChan:           make(chan struct{}, 1),
		timer:              timer,
	}
}

func (esw *eventStopWatch) fetch() []wire.Event {
	timer := time.NewTimer(esw.timer.Timeout)
	select {
	case <-timer.C:
		return nil
	case collectedVotes := <-esw.collectedVotesChan:
		timer.Stop()
		return collectedVotes
	case <-esw.stopChan:
		timer.Stop()
		return nil
	}
}

func (esw *eventStopWatch) stop() {
	select {
	case esw.stopChan <- empty:
	default:
	}
}

type reducer struct {
	firstStep   *eventStopWatch
	secondStep  *eventStopWatch
	ctx         *context
	accumulator *consensus.Accumulator

	sync.RWMutex
	stale bool

	publisher wire.EventPublisher
}

func newReducer(collectedVotesChan chan []wire.Event, ctx *context,
	publisher wire.EventPublisher, accumulator *consensus.Accumulator) *reducer {
	return &reducer{
		ctx:         ctx,
		firstStep:   newEventStopWatch(collectedVotesChan, ctx.timer),
		secondStep:  newEventStopWatch(collectedVotesChan, ctx.timer),
		publisher:   publisher,
		accumulator: accumulator,
	}
}

func (r *reducer) inCommittee() bool {
	round := r.ctx.state.Round()
	step := r.ctx.state.Step()
	return r.ctx.committee.AmMember(round, step)
}

func (r *reducer) startReduction(hash []byte) {
	log.Traceln("Starting Reduction")
	// empty out stop channels
	r.firstStep.stopChan = make(chan struct{}, 1)
	r.secondStep.stopChan = make(chan struct{}, 1)

	r.Lock()
	r.stale = false
	r.Unlock()

	if r.inCommittee() {
		r.sendReduction(bytes.NewBuffer(hash))
	}

	go r.begin()
}

func (r *reducer) begin() {
	log.WithField("process", "reducer").Traceln("Beginning Reduction")
	events := r.firstStep.fetch()
	log.WithField("process", "reducer").Traceln("First step completed")
	hash1 := r.extractHash(events)
	r.RLock()
	if !r.stale {
		// if there was a timeout, we should report nodes that did not vote
		if events == nil {
			_ = r.ctx.committee.ReportAbsentees(r.accumulator.All(),
				r.ctx.state.Round(), r.ctx.state.Step())
		}
		r.ctx.state.IncrementStep()
		if r.inCommittee() {
			r.sendReduction(hash1)
		}
	}
	r.RUnlock()

	eventsSecondStep := r.secondStep.fetch()
	log.WithField("process", "reducer").Traceln("Second step completed")
	r.RLock()
	defer r.RUnlock()
	if !r.stale {
		if eventsSecondStep == nil {
			_ = r.ctx.committee.ReportAbsentees(r.accumulator.All(),
				r.ctx.state.Round(), r.ctx.state.Step())
		}
		hash2 := r.extractHash(eventsSecondStep)
		allEvents := append(events, eventsSecondStep...)
		if r.isReductionSuccessful(hash1, hash2) {
			log.WithFields(log.Fields{
				"process":    "reducer",
				"votes":      len(allEvents),
				"block hash": hex.EncodeToString(hash1.Bytes()),
			}).Debugln("Reduction successful")

			r.sendResults(allEvents)
		}

		r.ctx.state.IncrementStep()
		r.publishRegeneration()
	}
}

func (r *reducer) sendReduction(hash *bytes.Buffer) {
	vote := new(bytes.Buffer)

	h := &header.Header{
		PubKeyBLS: r.ctx.Keys.BLSPubKeyBytes,
		Round:     r.ctx.state.Round(),
		Step:      r.ctx.state.Step(),
		BlockHash: hash.Bytes(),
	}

	if err := header.MarshalSignableVote(vote, h); err != nil {
		logErr(err, hash.Bytes(), "Error during marshalling of the reducer vote")
		return
	}

	if err := SignBuffer(vote, r.ctx.Keys); err != nil {
		logErr(err, hash.Bytes(), "Error while signing vote")
		return
	}

	message, err := wire.AddTopic(vote, topics.Reduction)
	if err != nil {
		logErr(err, hash.Bytes(), "Error while adding topic")
		return
	}

	r.publisher.Stream(string(topics.Gossip), message)
}

func (r *reducer) sendResults(events []wire.Event) {
	buf := new(bytes.Buffer)
	if err := r.ctx.handler.MarshalVoteSet(buf, events); err != nil {
		log.WithField("process", "reduction").WithError(err).Errorln("problem marshalling voteset")
		return
	}
	r.publisher.Publish(msg.ReductionResultTopic, buf)
}

func logErr(err error, hash []byte, msg string) {
	log.WithFields(
		log.Fields{
			"process":    "reducer",
			"block hash": hash,
		}).WithError(err).Errorln(msg)
}

func (r *reducer) extractHash(events []wire.Event) *bytes.Buffer {
	if events == nil {
		return bytes.NewBuffer(make([]byte, 32))
	}

	hash := new(bytes.Buffer)
	if err := r.ctx.handler.ExtractIdentifier(events[0], hash); err != nil {
		panic(err)
	}
	return hash
}

func (r *reducer) publishRegeneration() {
	roundAndStep := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundAndStep, r.ctx.state.Round())
	roundAndStep = append(roundAndStep, byte(r.ctx.state.Step()))
	r.publisher.Publish(msg.BlockRegenerationTopic, bytes.NewBuffer(roundAndStep))
}

func (r *reducer) isReductionSuccessful(hash1, hash2 *bytes.Buffer) bool {
	bothNotNil := !bytes.Equal(hash1.Bytes(), make([]byte, 32)) &&
		!bytes.Equal(hash2.Bytes(), make([]byte, 32))
	identicalResults := bytes.Equal(hash1.Bytes(), hash2.Bytes())
	return bothNotNil && identicalResults
}

func (r *reducer) end() {
	r.Lock()
	defer r.Unlock()
	r.stale = true
	r.firstStep.stop()
	r.secondStep.stop()
}
