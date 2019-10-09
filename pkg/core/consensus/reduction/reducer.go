package reduction

import (
	"bytes"
	"encoding/hex"
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
	case esw.stopChan <- struct{}{}:
	default:
	}
}

type reducer struct {
	firstStep   *eventStopWatch
	secondStep  *eventStopWatch
	accumulator *consensus.Accumulator

	publisher eventbus.Publisher
	signer    *consensus.Signer
	rpcBus    *rpcbus.RPCBus

	requestStepUpdate func()
}

func newReducer(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, timeOut time.Duration, requestStepUpdate func()) *reducer {
	return &reducer{
		publisher:         publisher,
		rpcBus:            rpcBus,
		timer:             consensus.NewTimer(timeOut),
		firstStep:         newEventStopWatch(ctx.timer),
		secondStep:        newEventStopWatch(ctx.timer),
		requestStepUpdate: requestStepUpdate,
	}
}

func (r *reducer) startReduction(hash []byte) {
	log.Traceln("Starting Reduction")
	// empty out stop channels
	r.firstStep.stopChan = make(chan struct{}, 1)
	r.secondStep.stopChan = make(chan struct{}, 1)

	r.sendReduction(hash)

	go r.begin()
}

func (r *reducer) begin() {
	log.WithField("process", "reducer").Traceln("Beginning Reduction")
	events := r.firstStep.fetch(r.accumulator.CollectedVotesChan)
	log.WithField("process", "reducer").Traceln("First step completed")
	hash := r.handleFirstResult(events)

	eventsSecondStep := r.secondStep.fetch(r.accumulator.CollectedVotesChan)
	log.WithField("process", "reducer").Traceln("Second step completed")
	r.handleSecondResult(events, eventsSecondStep, hash)
}

func (r *reducer) collectFirstStep(hdr header.Header, ev Reduction) {
	sv := r.aggregator.Aggregate(hdr, ev)
	if sv != nil {
		r.handleFirstResult(*sv, hdr.BlockHash)
	}
}

func (r *reducer) handleFirstResult(sv StepVotes, hash []byte) {
	r.requestStepUpdate()

	// If our result was not a zero value hash, we should first verify it
	// before voting on it again
	if !bytes.Equal(hash, emptyHash[:]) {
		req := rpcbus.NewRequest(*(bytes.NewBuffer(hash)), 5)
		if _, err := r.rpcBus.Call(rpcbus.VerifyCandidateBlock, req); err != nil {
			log.WithFields(log.Fields{
				"process": "reduction",
				"error":   err,
			}).Errorln("verifying the candidate block failed")
			r.sendReduction(emptyHash[:])
		}
	}

	r.sendReduction(hash)
}

func (r *reducer) finalizeSecondStep() {
	r.publishRegeneration()
	r.requestStepUpdate()
}

func (r *reducer) handleSecondResult(events, eventsSecondStep []wire.Event, hash1 []byte) {
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
	r.timer.IncreaseTimeOut()
}

func (r *reducer) GenerateReduction(hash []byte) (*bytes.Buffer, error) {
	sig, err := r.signer.BLSSign(hash)
	if err != nil {
		return nil, err
	}

	vote := new(bytes.Buffer)
	if err := encoding.Write256(vote, hash); err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(vote, sig); err != nil {
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
	if err := r.handler.ExtractIdentifier(events[0], hash); err != nil {
		panic(err)
	}
	return hash.Bytes()
}

func (r *reducer) publishRegeneration() {
	r.publisher.Publish(topics.BlockRegeneration, bytes.Buffer{})
}

func (r *reducer) isReductionSuccessful(hash1, hash2 []byte) bool {
	bothNotNil := !bytes.Equal(hash1, emptyHash[:]) && !bytes.Equal(hash2, emptyHash[:])
	return bothNotNil && bytes.Equal(hash1, hash2)
}

func (r *reducer) end() {
	r.firstStep.stop()
	r.secondStep.stop()
}
