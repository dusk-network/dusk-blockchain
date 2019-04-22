package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// LaunchNotification is a helper function allowing node internal processes interested in reduction messages to receive Reduction events as they get produced
func LaunchNotification(eventbus wire.EventSubscriber) <-chan *events.Reduction {
	revChan := make(chan *events.Reduction)
	evChan := consensus.LaunchNotification(eventbus, events.NewReductionUnMarshaller(), msg.OutgoingBlockReductionTopic)

	go func() {
		for {
			rEv := <-evChan
			reductionEvent := rEv.(*events.Reduction)
			revChan <- reductionEvent
		}
	}()

	return revChan
}

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

func (esw *eventStopWatch) reset() {
	for len(esw.stopChan) > 0 {
		<-esw.stopChan
	}
}

func (esw *eventStopWatch) stop() {
	esw.stopChan <- empty
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

func (r *reducer) startReduction(hash []byte) {
	log.Traceln("Starting Reduction")
	r.Lock()
	r.stale = false
	r.Unlock()
	r.sendReductionVote(bytes.NewBuffer(hash))
	r.firstStep.reset()
	r.secondStep.reset()
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
			r.ctx.committee.ReportAbsentees(r.accumulator.GetAllEvents(),
				r.ctx.state.Round(), r.ctx.state.Step())
		}
		r.ctx.state.IncrementStep()
		r.sendReductionVote(hash1)
	}
	r.RUnlock()

	eventsSecondStep := r.secondStep.fetch()
	log.WithField("process", "reducer").Traceln("Second step completed")
	r.RLock()
	defer r.RUnlock()
	if !r.stale {
		if eventsSecondStep == nil {
			r.ctx.committee.ReportAbsentees(r.accumulator.GetAllEvents(),
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

	_, _ = buffer.Write(hash.Bytes())
	return buffer, nil
}
