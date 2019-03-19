package reduction

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type sigSetReduction struct {
	*reductionEvent
	winningBlockHash []byte
}

func (ssr *sigSetReduction) Equal(e wire.Event) bool {
	return ssr.reductionEvent.Equal(e) &&
		bytes.Equal(ssr.winningBlockHash, e.(*sigSetReduction).winningBlockHash)
}

type sigSetReductionUnmarshaller struct {
	*reductionEventUnmarshaller
}

func newSigSetReductionUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return &reductionEventUnmarshaller{validate}
}

func (ssru *sigSetReductionUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetReduction := e.(*sigSetReduction)

	if err := ssru.reductionEventUnmarshaller.Unmarshal(r, sigSetReduction.reductionEvent); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &sigSetReduction.winningBlockHash); err != nil {
		return err
	}

	return nil
}

type sigSetReductionCollector struct {
	*reductionCollector
	unmarshaller wire.EventUnmarshaller

	reducing         bool
	winningBlockHash []byte
}

func newSigSetReductionCollector(committee user.Committee, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error,
	hashChannel, resultChannel chan *bytes.Buffer) *sigSetReductionCollector {

	reductionCollector := newReductionCollector(committee, timerLength, validateFunc,
		hashChannel, resultChannel)

	sigSetReductionCollector := &sigSetReductionCollector{
		reductionCollector: reductionCollector,
		unmarshaller:       newSigSetReductionUnmarshaller(validateFunc),
	}

	return sigSetReductionCollector
}

func (ssrc *sigSetReductionCollector) updateRound(round uint64) {
	ssrc.winningBlockHash = nil
	ssrc.currentRound = round
}

func (ssrc sigSetReductionCollector) correctWinningBlockHash(m *sigSetReduction) bool {
	return bytes.Equal(m.winningBlockHash, ssrc.winningBlockHash)
}

func (ssrc *sigSetReductionCollector) Collect(buffer *bytes.Buffer) error {
	event := &sigSetReduction{}
	if err := ssrc.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	ssrc.process(event)
	return nil
}

func (ssrc *sigSetReductionCollector) process(m *sigSetReduction) {
	if ssrc.shouldBeProcessed(m.reductionEvent) && blsVerified(m.reductionEvent) &&
		ssrc.winningBlockHash != nil && ssrc.correctWinningBlockHash(m) {

		if !ssrc.reducing {
			ssrc.reducing = true
			go ssrc.runReduction()
		}

		ssrc.inputChannel <- m.reductionEvent
	} else if ssrc.shouldBeStored(m.reductionEvent) && blsVerified(m.reductionEvent) {
		ssrc.queue.PutMessage(m.Round, m.Step, m)
	}
}

type SigSetReducer struct {
	eventBus              *wire.EventBus
	reductionSubscriber   *wire.EventSubscriber
	phaseUpdateSubscriber *wire.EventSubscriber
	voteSubscriber        *wire.EventSubscriber
	roundSubscriber       *wire.EventSubscriber
	*sigSetReductionCollector

	// channels linked to subscribers
	voteChannel        <-chan []byte
	phaseUpdateChannel <-chan []byte
	roundChannel       <-chan uint64

	// channels linked to the reductioncollector
	hashChannel   <-chan *bytes.Buffer
	resultChannel <-chan *bytes.Buffer
}

func NewSigSetReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee user.Committee, timerLength time.Duration) *SigSetReducer {

	voteChannel := make(chan []byte, 1)
	phaseUpdateChannel := make(chan []byte, 1)
	roundChannel := make(chan uint64, 1)

	hashChannel := make(chan *bytes.Buffer, 1)
	resultChannel := make(chan *bytes.Buffer, 1)

	reductionCollector := newSigSetReductionCollector(committee, timerLength,
		validateFunc, hashChannel, resultChannel)
	reductionSubscriber := wire.NewEventSubscriber(eventBus, reductionCollector,
		string(topics.BlockReduction))

	phaseUpdateCollector := &phaseUpdateCollector{phaseUpdateChannel}
	phaseUpdateSubscriber := wire.NewEventSubscriber(eventBus, phaseUpdateCollector,
		string(msg.PhaseUpdateTopic))

	voteCollector := &voteCollector{voteChannel}
	voteSubscriber := wire.NewEventSubscriber(eventBus, voteCollector, string(msg.SelectionResultTopic))

	roundCollector := &roundCollector{roundChannel}
	roundSubscriber := wire.NewEventSubscriber(eventBus, roundCollector, string(msg.RoundUpdateTopic))

	sigSetReducer := &SigSetReducer{
		eventBus:                 eventBus,
		reductionSubscriber:      reductionSubscriber,
		phaseUpdateSubscriber:    phaseUpdateSubscriber,
		voteSubscriber:           voteSubscriber,
		roundSubscriber:          roundSubscriber,
		sigSetReductionCollector: reductionCollector,
		voteChannel:              voteChannel,
		roundChannel:             roundChannel,
		hashChannel:              hashChannel,
		resultChannel:            resultChannel,
	}

	return sigSetReducer
}

func (sr SigSetReducer) vote(hash []byte, round uint64, step uint8) error {
	if !sr.voted && sr.winningBlockHash != nil {
		buffer := new(bytes.Buffer)

		if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
			return err
		}

		if err := encoding.WriteUint8(buffer, step); err != nil {
			return err
		}

		if err := encoding.Write256(buffer, hash); err != nil {
			return err
		}

		if err := encoding.Write256(buffer, sr.winningBlockHash); err != nil {
			return err
		}

		sr.eventBus.Publish(msg.OutgoingReductionTopic, buffer)
		sr.voted = true
	}

	return nil
}

func (sr *SigSetReducer) Listen() {
	go sr.reductionSubscriber.Accept()
	go sr.phaseUpdateSubscriber.Accept()
	go sr.voteSubscriber.Accept()
	go sr.roundSubscriber.Accept()

	for {
		select {
		case sigSetHash := <-sr.voteChannel:
			if err := sr.vote(sigSetHash, sr.currentRound, sr.currentStep); err != nil {
				// Log
				return
			}
		case reductionVote := <-sr.hashChannel:
			sr.eventBus.Publish(msg.OutgoingReductionTopic, reductionVote)
		case result := <-sr.resultChannel:
			sr.eventBus.Publish(msg.OutgoingAgreementTopic, result)
		default:
			if sr.winningBlockHash != nil {
				sr.checkQueue()
			}

			if !sr.reducing {
				sr.checkRoundChannel()
			}
		}
	}
}

func (sr SigSetReducer) checkQueue() {
	queuedMessages := sr.queue.GetMessages(sr.currentRound, sr.currentStep)

	if queuedMessages != nil {
		for _, message := range queuedMessages {
			m := message.(*sigSetReduction)
			sr.process(m)
		}
	}
}

func (sr *SigSetReducer) checkRoundChannel() {
	select {
	case round := <-sr.roundChannel:
		sr.updateRound(round)
	default:
		break
	}
}

type phaseUpdateCollector struct {
	phaseUpdateChannel chan<- []byte
}

func (p *phaseUpdateCollector) Collect(phaseUpdateBuffer *bytes.Buffer) error {
	p.phaseUpdateChannel <- phaseUpdateBuffer.Bytes()
	return nil
}
