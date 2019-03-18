package reduction

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

type blockReduction = reductionEvent

type blockReductionCollector struct {
	*reductionCollector
	hashChannel   chan<- []byte
	resultChannel chan<- []byte
	unmarshaller  EventUnmarshaller

	reducing bool
}

func newBlockReductionCollector(committee user.Committee, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error, hashChannel, resultChannel chan []byte) *blockReductionCollector {

	reductionCollector := newReductionCollector(committee, hashChannel,
		resultChannel, timerLength, validateFunc)

	blockReductionCollector := &blockReductionCollector{
		reductionCollector: reductionCollector,
		hashChannel:        hashChannel,
		resultChannel:      resultChannel,
		unmarshaller:       newReductionEventUnmarshaller(validateFunc),
	}

	return blockReductionCollector
}

func (brc *blockReductionCollector) updateRound(round uint64) {
	brc.currentRound = round
}

func (brc *blockReductionCollector) Collect(buffer *bytes.Buffer) error {
	event := &reductionEvent{}
	if err := brc.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	brc.process(event)
	return nil
}

func (brc *blockReductionCollector) process(m *blockReduction) {
	if brc.shouldBeProcessed(m) && blsVerified(m) {
		if !brc.reducing {
			brc.reducing = true
			go brc.runReduction()
		}

		brc.inputChannel <- m
	} else if brc.shouldBeStored(m) && blsVerified(m) {
		brc.queue.PutMessage(m.Round, m.Step, m)
	}
}

type blockReducer struct {
	eventBus        *wire.EventBus
	hashSubscriber  *EventSubscriber
	roundSubscriber *EventSubscriber
	*blockReductionCollector

	hashChannel   <-chan []byte
	resultChannel <-chan []byte
	roundChannel  <-chan uint64
}

func newBlockReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee user.Committee, timerLength time.Duration) *blockReducer {

	hashChannel := make(chan []byte, 100)
	roundChannel := make(chan uint64, 1)
	resultChannel := make(chan []byte, 1)

	reductionCollector := newBlockReductionCollector(committee, timerLength,
		validateFunc, hashChannel, resultChannel)
	hashSubscriber := NewEventSubscriber(eventBus, reductionCollector,
		string(topics.BlockReduction))

	roundCollector := &RoundCollector{roundChannel}
	roundSubscriber := NewEventSubscriber(eventBus, roundCollector, string(msg.RoundUpdateTopic))

	blockReducer := &blockReducer{
		eventBus:                eventBus,
		hashSubscriber:          hashSubscriber,
		roundSubscriber:         roundSubscriber,
		blockReductionCollector: reductionCollector,
	}

	return blockReducer
}

func (br blockReducer) vote(hash []byte, round uint64, step uint8) error {
	if !br.voted {
		buffer := bytes.NewBuffer(hash)

		if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
			return err
		}

		if err := encoding.WriteUint8(buffer, step); err != nil {
			return err
		}

		br.eventBus.Publish(msg.OutgoingReductionTopic, buffer)
		br.voted = true
	}

	return nil
}

func (br *blockReducer) Listen() {
	go br.hashSubscriber.Accept()
	go br.roundSubscriber.Accept()

	for {
		select {
		case blockHash := <-br.hashChannel:
			if err := br.vote(blockHash, br.currentRound, br.currentStep); err != nil {
				// Log
				return
			}
		case result := <-br.resultChannel:
			buffer := bytes.NewBuffer(result)
			br.eventBus.Publish(msg.OutgoingAgreementTopic, buffer)
		default:
			br.checkQueue()

			if !br.reducing {
				br.checkRoundChannel()
			}
		}
	}
}

func (br blockReducer) checkQueue() {
	queuedMessages := br.queue.GetMessages(br.currentRound, br.currentStep)

	if queuedMessages != nil {
		for _, message := range queuedMessages {
			br.process(message)
		}
	}
}

func (br *blockReducer) checkRoundChannel() {
	select {
	case round := <-br.roundChannel:
		br.updateRound(round)
	default:
		break
	}
}

// RoundCollector is a simple wrapper over a channel to get round notifications
type RoundCollector struct {
	roundChan chan<- uint64
}

// Collect as specified in the EventCollector interface. In this case Collect simply performs unmarshalling of the round event
func (r *RoundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}
