package reduction

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// Reducer is a collection of common fields and methods, both used by the
	// BlockReducer and SigSetReducer.
	Reducer struct {
		eventBus         *wire.EventBus
		currentSequencer *reductionSequencer
		unmarshaller     wire.EventUnmarshaller

		*reductionEventCollector
		committee    committee.Committee
		currentRound uint64
		currentStep  uint8
		timeOut      time.Duration
		queue        *consensus.EventQueue
		stopChannel  chan bool

		// channels linked to subscribers
		roundChannel       <-chan uint64
		phaseUpdateChannel <-chan []byte
	}
)

// newReducer will create a new standard reducer and return it.
func newReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timeOut time.Duration) *Reducer {

	queue := consensus.NewEventQueue()
	roundCollector := consensus.InitRoundCollector(eventBus)
	phaseUpdateCollector := consensus.InitPhaseCollector(eventBus)

	return &Reducer{
		eventBus:           eventBus,
		unmarshaller:       newReductionEventUnmarshaller(validateFunc),
		roundChannel:       roundCollector.RoundChan,
		phaseUpdateChannel: phaseUpdateCollector.BlockHashChan,
		queue:              &queue,
		committee:          committee,
		timeOut:            timeOut,
	}
}

func (r *Reducer) process(m *Event) {
	if r.shouldBeProcessed(m) {
		r.currentSequencer.incomingReductionChannel <- m
		r.Store(m, m.PubKeyBLS)
	} else if r.shouldBeStored(m) {
		r.queue.PutEvent(m.Round, m.Step, m)
	}
}

func (r Reducer) processQueuedMessages() {
	queuedEvents := r.queue.GetEvents(r.currentRound, r.currentStep)
	if queuedEvents != nil {
		for _, event := range queuedEvents {
			ev := event.(*Event)
			r.process(ev)
		}
	}
}

func (r *Reducer) updateRound(round uint64) {
	r.queue.Clear(r.currentRound)
	r.currentRound = round
	r.currentStep = 1
	r.currentSequencer = nil

	r.Clear()
}

func (r *Reducer) incrementStep() {
	r.currentStep++
	r.Clear()
}

func (r Reducer) shouldBeProcessed(m *Event) bool {
	isDupe := r.Contains(m.PubKeyBLS)
	isMember := r.committee.IsMember(m.PubKeyBLS)
	isRelevant := r.currentRound == m.Round && r.currentStep == m.Step

	return !isDupe && isMember && isRelevant && r.currentSequencer != nil
}

func (r Reducer) shouldBeStored(m *Event) bool {
	futureRound := r.currentRound <= m.Round
	futureStep := r.currentStep <= m.Step

	return futureRound || futureStep
}

func (r Reducer) addVoteInfo(data []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, r.currentRound); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, r.currentStep); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(data); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (r *Reducer) startSequencer() {
	r.currentSequencer = newReductionSequencer(r.committee.Quorum(), r.timeOut)
	go r.currentSequencer.startSequence()
	go r.listenSequencer()
	r.processQueuedMessages()
}

func (r *Reducer) listenSequencer() {
	for {
		select {
		case <-r.stopChannel:
			return
		case hash := <-r.currentSequencer.outgoingReductionChannel:
			r.vote(hash, msg.OutgoingReductionTopic)
			r.incrementStep()
			r.processQueuedMessages()
		case hashAndVoteSet := <-r.currentSequencer.outgoingAgreementChannel:
			r.vote(hashAndVoteSet, msg.OutgoingAgreementTopic)
			r.currentSequencer = nil
			r.incrementStep()
			return
		}
	}
}

func (r *Reducer) stopSequencer() {
	if r.currentSequencer != nil {
		r.stopChannel <- true
		r.currentSequencer = nil
	}
}

func (r Reducer) vote(data []byte, topic string) error {
	voteData, err := r.addVoteInfo(data)
	if err != nil {
		return err
	}

	r.eventBus.Publish(topic, voteData)
	return nil
}
