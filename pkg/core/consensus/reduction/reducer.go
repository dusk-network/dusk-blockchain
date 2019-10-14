package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var emptyHash = [32]byte{}

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

	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus
}

func newReducer(ctx *context, publisher eventbus.Publisher, filter *consensus.EventFilter, rpcBus *rpcbus.RPCBus) *reducer {
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
	r.lock.Lock()
	defer r.lock.Unlock()
	r.firstStep.stopChan = make(chan struct{}, 1)
	r.secondStep.stopChan = make(chan struct{}, 1)

	r.stale = false

	if r.inCommittee() {
		r.sendReduction(hash)
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

func (r *reducer) handleFirstResult(events []wire.Event) []byte {
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
		if !bytes.Equal(hash, emptyHash[:]) {
			req := rpcbus.NewRequest(*(bytes.NewBuffer(hash)))
			if _, err := r.rpcBus.Call(rpcbus.VerifyCandidateBlock, req, 5); err != nil {
				log.WithFields(log.Fields{
					"process": "reduction",
					"error":   err,
				}).Errorln("verifying the candidate block failed")
				r.sendReduction(emptyHash[:])
				return hash
			}
		}

		r.sendReduction(hash)
		return hash
	}

	return nil
}

func (r *reducer) finalizeSecondStep() {
	r.publishRegeneration()
	r.ctx.state.IncrementStep()
	r.filter.ResetAccumulator()
}

func (r *reducer) handleSecondResult(events, eventsSecondStep []wire.Event, hash1 []byte) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if !r.stale {
		defer r.finalizeSecondStep()

		hash2 := r.extractHash(eventsSecondStep)
		if r.isReductionSuccessful(hash1, hash2) {
			allEvents := append(events, eventsSecondStep...)
			log.WithFields(log.Fields{
				"process":    "reducer",
				"votes":      len(allEvents),
				"block hash": hex.EncodeToString(hash1),
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

func (r *reducer) GenerateReduction(hash []byte) (*bytes.Buffer, error) {
	vote := new(bytes.Buffer)

	h := &header.Header{
		PubKeyBLS: r.ctx.handler.Keys.BLSPubKeyBytes,
		Round:     r.ctx.state.Round(),
		Step:      r.ctx.state.Step(),
		BlockHash: hash,
	}

	if err := header.MarshalSignableVote(vote, h); err != nil {
		return nil, err
	}

	if err := SignBuffer(vote, r.ctx.handler.Keys); err != nil {
		return nil, err
	}

	// we need to prepend the topic AFTER signing the buffer otherwise it'd influence the hashing
	if err := topics.Prepend(vote, topics.Reduction); err != nil {
		return nil, err
	}

	return vote, nil
}

func (r *reducer) sendReduction(hash []byte) {
	vote, err := r.GenerateReduction(hash)
	if err != nil {
		logErr(err, hash, "Error while generating reducer vote")
		return
	}

	r.publisher.Publish(topics.Gossip, vote)
}

func (r *reducer) sendResults(events []wire.Event) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, r.ctx.state.Round()); err != nil {
		panic(err)
	}

	if events != nil {
		if err := r.ctx.handler.MarshalVoteSet(buf, events); err != nil {
			log.WithField("process", "reduction").WithError(err).Errorln("problem marshalling voteset")
			return
		}
	}
	r.publisher.Publish(topics.ReductionResult, buf)
}

func logErr(err error, hash []byte, msg string) {
	log.WithFields(
		log.Fields{
			"process":    "reducer",
			"block hash": hash,
		}).WithError(err).Errorln(msg)
}

func (r *reducer) extractHash(events []wire.Event) []byte {
	if events == nil {
		return emptyHash[:]
	}

	hash := new(bytes.Buffer)
	if err := r.ctx.handler.ExtractIdentifier(events[0], hash); err != nil {
		panic(err)
	}
	return hash.Bytes()
}

func (r *reducer) publishRegeneration() {
	roundAndStep := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundAndStep, r.ctx.state.Round())
	roundAndStep = append(roundAndStep, byte(r.ctx.state.Step()))
	r.publisher.Publish(topics.BlockRegeneration, bytes.NewBuffer(roundAndStep))
}

func (r *reducer) isReductionSuccessful(hash1, hash2 []byte) bool {
	bothNotNil := !bytes.Equal(hash1, emptyHash[:]) && !bytes.Equal(hash2, emptyHash[:])
	return bothNotNil && bytes.Equal(hash1, hash2)
}

func (r *reducer) end() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.stale = true
	r.firstStep.stop()
	r.secondStep.stop()
}
