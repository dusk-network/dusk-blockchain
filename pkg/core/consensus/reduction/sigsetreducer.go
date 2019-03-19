package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type SigSetReduction struct {
	*Event
	winningBlockHash []byte
}

func (ssr *SigSetReduction) Equal(e wire.Event) bool {
	return ssr.Event.Equal(e) &&
		bytes.Equal(ssr.winningBlockHash, e.(*SigSetReduction).winningBlockHash)
}

type sigSetReductionUnmarshaller struct {
	*reductionEventUnmarshaller
}

func newSigSetReductionUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return &reductionEventUnmarshaller{validate}
}

func (ssru *sigSetReductionUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetReduction := e.(*SigSetReduction)

	if err := ssru.reductionEventUnmarshaller.Unmarshal(r, sigSetReduction.Event); err != nil {
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

func newSigSetReductionCollector(committee committee.Committee, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error) *sigSetReductionCollector {

	reductionCollector := newReductionCollector(committee, timerLength)

	sigSetReductionCollector := &sigSetReductionCollector{
		reductionCollector: reductionCollector,
		unmarshaller:       newSigSetReductionUnmarshaller(validateFunc),
	}

	return sigSetReductionCollector
}

func (ssrc *sigSetReductionCollector) updateRound(round uint64) {
	ssrc.queue.Clear(ssrc.currentRound)
	ssrc.winningBlockHash = nil
	ssrc.currentRound = round
	ssrc.currentStep = 1

	if ssrc.reducing {
		ssrc.stopSelector()
		ssrc.reducing = false
	}

	ssrc.Clear()
}

func (ssrc sigSetReductionCollector) correctWinningBlockHash(m *SigSetReduction) bool {
	return bytes.Equal(m.winningBlockHash, ssrc.winningBlockHash)
}

func (ssrc *sigSetReductionCollector) Collect(buffer *bytes.Buffer) error {
	event := &SigSetReduction{}
	if err := ssrc.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	ssrc.process(event)
	return nil
}

func (ssrc *sigSetReductionCollector) process(m *SigSetReduction) {
	if ssrc.shouldBeProcessed(m.Event) && blsVerified(m.Event) &&
		ssrc.winningBlockHash != nil && ssrc.correctWinningBlockHash(m) {

		if !ssrc.reducing {
			ssrc.reducing = true
			go ssrc.startSelector()
		}

		ssrc.incomingReductionChannel <- m.Event
		pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
		ssrc.Store(m, pubKeyStr)
	} else if ssrc.shouldBeStored(m.Event) && blsVerified(m.Event) {
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
}

func NewSigSetReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timerLength time.Duration) *SigSetReducer {

	voteChannel := make(chan []byte, 1)
	phaseUpdateChannel := make(chan []byte, 1)
	roundChannel := make(chan uint64, 1)

	reductionCollector := newSigSetReductionCollector(committee, timerLength,
		validateFunc)
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
	}

	return sigSetReducer
}

func (sr SigSetReducer) addVoteInfo(data []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, sr.currentRound); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, sr.currentStep); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(data); err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, sr.winningBlockHash); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (sr *SigSetReducer) Listen() {
	go sr.reductionSubscriber.Accept()
	go sr.phaseUpdateSubscriber.Accept()
	go sr.voteSubscriber.Accept()
	go sr.roundSubscriber.Accept()

	for {
		select {
		case sigSetHash := <-sr.voteChannel:
			if !sr.votedThisStep {
				vote, err := sr.addVoteInfo(sigSetHash)
				if err != nil {
					// Log
					return
				}

				sr.eventBus.Publish(msg.OutgoingReductionTopic, vote)
				sr.votedThisStep = true
			}
		case sigSetHash := <-sr.hashChannel:
			sr.incrementStep()
			vote, err := sr.addVoteInfo(sigSetHash)
			if err != nil {
				// Log
				return
			}

			sr.eventBus.Publish(msg.OutgoingReductionTopic, vote)
			sr.votedThisStep = true
		case result := <-sr.resultChannel:
			if result != nil {
				vote, err := sr.addVoteInfo(result)
				if err != nil {
					// Log
					return
				}

				sr.eventBus.Publish(msg.OutgoingAgreementTopic, vote)
			}

			sr.incrementStep()
			sr.reducing = false
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
			m := message.(*SigSetReduction)
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
