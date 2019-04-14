package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

var empty struct{}

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
		esw.timer.TimeoutChan <- empty
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

	sync.RWMutex
	stale bool

	publisher wire.EventPublisher
}

func newReducer(ctx *context, publisher wire.EventPublisher) *reducer {
	return &reducer{
		ctx:       ctx,
		publisher: publisher,
	}
}

func (r *reducer) isStale() bool {
	r.RLock()
	defer r.RUnlock()
	return r.stale
}

// There is no mutex involved here, as this function is only ever called by the broker,
// who does it synchronously. The reducer variable therefore can not be in a race
// condition with another goroutine.
func (r *reducer) startReduction(collectedVotesChan chan []wire.Event, hash []byte) {
	log.Traceln("Starting Reduction")
	r.sendReductionVote(bytes.NewBuffer(hash))
	r.firstStep = newEventStopWatch(collectedVotesChan, r.ctx.timer)
	r.secondStep = newEventStopWatch(collectedVotesChan, r.ctx.timer)
	go r.begin()
}

func (r *reducer) begin() {
	log.WithField("process", "reducer").Traceln("Beginning Reduction")
	events := r.firstStep.fetch()
	log.WithField("process", "reducer").Traceln("First step completed")
	hash1 := r.extractHash(events)
	if !r.isStale() {
		r.ctx.state.IncrementStep()
		r.sendReductionVote(hash1)
	}

	eventsSecondStep := r.secondStep.fetch()
	log.WithField("process", "reducer").Traceln("Second step completed")
	if !r.isStale() {
		hash2 := r.extractHash(eventsSecondStep)
		allEvents := append(events, eventsSecondStep...)
		if r.isReductionSuccessful(hash1, hash2) {
			log.WithFields(log.Fields{
				"process":    "reducer",
				"votes":      len(allEvents),
				"block hash": hex.EncodeToString(hash1.Bytes()),
			}).Debugln("Reduction successful")
			r.sendAgreementVote(allEvents, hash2)
		}

		r.ctx.state.IncrementStep()
		r.publishRegeneration()
	}
}

func (r *reducer) sendReductionVote(hash *bytes.Buffer) {
	vote, err := r.marshalHeader(hash)
	if err != nil {
		panic(err)
	}
	r.publisher.Publish(msg.OutgoingBlockReductionTopic, vote)
}

func (r *reducer) sendAgreementVote(events []wire.Event, hash *bytes.Buffer) {
	if err := r.ctx.handler.MarshalVoteSet(hash, events); err != nil {
		panic(err)
	}
	agreementVote, err := r.marshalHeader(hash)
	if err != nil {
		panic(err)
	}
	r.publisher.Publish(msg.OutgoingBlockAgreementTopic, agreementVote)
}

func (r *reducer) publishRegeneration() {
	roundAndStep := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundAndStep, r.ctx.state.Round())
	roundAndStep = append(roundAndStep, byte(r.ctx.state.Step()))
	r.publisher.Publish(msg.BlockRegenerationTopic, bytes.NewBuffer(roundAndStep))
}

func (r *reducer) isReductionSuccessful(hash1, hash2 *bytes.Buffer) bool {
	bothNotNil := hash1 != nil && hash2 != nil
	identicalResults := bytes.Equal(hash1.Bytes(), hash2.Bytes())
	return bothNotNil && identicalResults
}

func (r *reducer) end() {
	r.Lock()
	r.stale = true
	r.Unlock()
	if r.firstStep != nil {
		r.firstStep.stop()
	}
	if r.secondStep != nil {
		r.secondStep.stop()
	}
	r.stale = false
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

func (r *reducer) marshalHeader(hash *bytes.Buffer) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	// Decoding Round
	if err := encoding.WriteUint64(buffer, binary.LittleEndian, r.ctx.state.Round()); err != nil {
		return nil, err
	}

	// Decoding Step
	if err := encoding.WriteUint8(buffer, r.ctx.state.Step()); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(hash.Bytes()); err != nil {
		return nil, err
	}
	return buffer, nil
}
