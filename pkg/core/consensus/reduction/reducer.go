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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var empty struct{}

type eventStopWatch struct {
	stopChan chan struct{}
	timer    *consensus.Timer
}

func newEventStopWatch(timer *consensus.Timer) *eventStopWatch {
	return &eventStopWatch{
		stopChan: make(chan struct{}, 1),
		timer:    timer,
	}
}

func (esw *eventStopWatch) fetch(collectedVotesChan chan []wire.Event) []wire.Event {
	timer := time.NewTimer(esw.timer.TimeOut())
	select {
	case <-timer.C:
		return nil
	case collectedVotes := <-collectedVotesChan:
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
	firstStep  *eventStopWatch
	secondStep *eventStopWatch
	ctx        *context
	filter     *consensus.EventFilter

	lock  sync.RWMutex
	stale bool

	publisher wire.EventPublisher
	rpcBus    *wire.RPCBus
}

func newReducer(ctx *context, publisher wire.EventPublisher, filter *consensus.EventFilter, rpcBus *wire.RPCBus) *reducer {
	return &reducer{
		ctx:        ctx,
		publisher:  publisher,
		firstStep:  newEventStopWatch(ctx.timer),
		secondStep: newEventStopWatch(ctx.timer),
		filter:     filter,
		rpcBus:     rpcBus,
	}
}

func (r *reducer) inCommittee() bool {
	round := r.ctx.state.Round()
	step := r.ctx.state.Step()
	return r.ctx.handler.AmMember(round, step)
}

func (r *reducer) startReduction(hash []byte) {
	log.Traceln("Starting Reduction")
	// empty out stop channels
	r.firstStep.stopChan = make(chan struct{}, 1)
	r.secondStep.stopChan = make(chan struct{}, 1)

	r.lock.Lock()
	r.stale = false
	r.lock.Unlock()

	if r.inCommittee() {
		r.sendReduction(bytes.NewBuffer(hash))
	}

	go r.begin()
}

func (r *reducer) begin() {
	log.WithField("process", "reducer").Traceln("Beginning Reduction")
	events := r.firstStep.fetch(r.filter.Accumulator.CollectedVotesChan)
	log.WithField("process", "reducer").Traceln("First step completed")
	hash := r.handleFirstResult(events)

	eventsSecondStep := r.secondStep.fetch(r.filter.Accumulator.CollectedVotesChan)
	log.WithField("process", "reducer").Traceln("Second step completed")
	r.handleSecondResult(events, eventsSecondStep, hash)
}

func (r *reducer) handleFirstResult(events []wire.Event) *bytes.Buffer {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if !r.stale {
		r.ctx.state.IncrementStep()
		r.filter.ResetAccumulator()
		r.filter.FlushQueue()

		hash := r.extractHash(events)
		if !r.inCommittee() {
			return hash
		}

		// If our result was not a zero value hash, we should first verify it
		// before voting on it again
		if !bytes.Equal(hash.Bytes(), make([]byte, 32)) {
			req := wire.NewRequest(*hash, 5)
			if _, err := r.rpcBus.Call(wire.VerifyCandidateBlock, req); err != nil {
				log.WithFields(log.Fields{
					"process": "reduction",
					"error":   err,
				}).Errorln("verifying the candidate block failed")
				r.sendReduction(bytes.NewBuffer(make([]byte, 32)))
				return hash
			}
		}

		r.sendReduction(hash)
		return hash
	}

	return nil
}

func (r *reducer) handleSecondResult(events, eventsSecondStep []wire.Event, hash1 *bytes.Buffer) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if !r.stale {
		defer r.filter.ResetAccumulator()
		defer r.ctx.state.IncrementStep()
		defer r.publishRegeneration()

		hash2 := r.extractHash(eventsSecondStep)
		if r.isReductionSuccessful(hash1, hash2) {
			allEvents := append(events, eventsSecondStep...)
			log.WithFields(log.Fields{
				"process":    "reducer",
				"votes":      len(allEvents),
				"block hash": hex.EncodeToString(hash1.Bytes()),
			}).Debugln("Reduction successful")

			r.sendResults(allEvents)
			return
		}

		// If we did not get a successful result, we still send a message to the
		// agreement component so that it stays synced with everyone else.
		r.sendResults(nil)
		r.ctx.timer.IncreaseTimeOut()
	}
}

func (r *reducer) sendReduction(hash *bytes.Buffer) {
	vote := new(bytes.Buffer)

	h := &header.Header{
		PubKeyBLS: r.ctx.handler.Keys.BLSPubKeyBytes,
		Round:     r.ctx.state.Round(),
		Step:      r.ctx.state.Step(),
		BlockHash: hash.Bytes(),
	}

	if err := header.MarshalSignableVote(vote, h); err != nil {
		logErr(err, hash.Bytes(), "Error during marshalling of the reducer vote")
		return
	}

	if err := SignBuffer(vote, r.ctx.handler.Keys); err != nil {
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
	if err := encoding.WriteUint64(buf, binary.LittleEndian, r.ctx.state.Round()); err != nil {
		panic(err)
	}

	if events != nil {
		if err := r.ctx.handler.MarshalVoteSet(buf, events); err != nil {
			log.WithField("process", "reduction").WithError(err).Errorln("problem marshalling voteset")
			return
		}
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
	r.lock.Lock()
	defer r.lock.Unlock()
	r.stale = true
	r.firstStep.stop()
	r.secondStep.stop()
}
