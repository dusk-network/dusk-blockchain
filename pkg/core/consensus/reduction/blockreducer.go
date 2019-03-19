package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type BlockReduction = Event

type blockReductionCollector struct {
	*reductionCollector
	unmarshaller wire.EventUnmarshaller

	reducing bool
}

func newBlockReductionCollector(committee committee.Committee, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error) *blockReductionCollector {

	reductionCollector := newReductionCollector(committee, timerLength)

	blockReductionCollector := &blockReductionCollector{
		reductionCollector: reductionCollector,
		unmarshaller:       newReductionEventUnmarshaller(validateFunc),
	}

	return blockReductionCollector
}

func (brc *blockReductionCollector) updateRound(round uint64) {
	brc.queue.Clear(brc.currentRound)
	brc.currentRound = round
	brc.currentStep = 1

	if brc.reducing {
		brc.stopSelector()
		brc.reducing = false
	}

	brc.Clear()
}

func (brc *blockReductionCollector) Collect(buffer *bytes.Buffer) error {
	event := &Event{}
	if err := brc.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	brc.process(event)
	return nil
}

func (brc *blockReductionCollector) process(m *BlockReduction) {
	if brc.shouldBeProcessed(m) && blsVerified(m) {
		if !brc.reducing {
			brc.reducing = true
			go brc.startSelector()
		}

		brc.incomingReductionChannel <- m
		pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
		brc.Store(m, pubKeyStr)
	} else if brc.shouldBeStored(m) && blsVerified(m) {
		brc.queue.PutMessage(m.Round, m.Step, m)
	}
}

type BlockReducer struct {
	eventBus            *wire.EventBus
	reductionSubscriber *wire.EventSubscriber
	voteSubscriber      *wire.EventSubscriber
	roundSubscriber     *wire.EventSubscriber
	*blockReductionCollector

	// channels linked to subscribers
	voteChannel  <-chan []byte
	roundChannel <-chan uint64
}

func NewBlockReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timerLength time.Duration) *BlockReducer {

	voteChannel := make(chan []byte, 1)
	roundChannel := make(chan uint64, 1)

	reductionCollector := newBlockReductionCollector(committee, timerLength,
		validateFunc)
	reductionSubscriber := wire.NewEventSubscriber(eventBus, reductionCollector,
		string(topics.BlockReduction))

	voteCollector := &voteCollector{voteChannel}
	voteSubscriber := wire.NewEventSubscriber(eventBus, voteCollector,
		string(msg.SelectionResultTopic))

	roundCollector := &roundCollector{roundChannel}
	roundSubscriber := wire.NewEventSubscriber(eventBus, roundCollector,
		string(msg.RoundUpdateTopic))

	blockReducer := &BlockReducer{
		eventBus:                eventBus,
		reductionSubscriber:     reductionSubscriber,
		voteSubscriber:          voteSubscriber,
		roundSubscriber:         roundSubscriber,
		blockReductionCollector: reductionCollector,
		voteChannel:             voteChannel,
		roundChannel:            roundChannel,
	}

	return blockReducer
}

func (br BlockReducer) addVoteInfo(data []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, br.currentRound); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, br.currentStep); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(data); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (br *BlockReducer) Listen() {
	go br.reductionSubscriber.Accept()
	go br.voteSubscriber.Accept()
	go br.roundSubscriber.Accept()

	for {
		select {
		case blockHash := <-br.voteChannel:
			if !br.votedThisStep {
				vote, err := br.addVoteInfo(blockHash)
				if err != nil {
					// Log
					return
				}

				br.eventBus.Publish(msg.OutgoingReductionTopic, vote)
				br.votedThisStep = true
			}
		case blockHash := <-br.hashChannel:
			br.incrementStep()
			vote, err := br.addVoteInfo(blockHash)
			if err != nil {
				// Log
				return
			}

			br.eventBus.Publish(msg.OutgoingReductionTopic, vote)
			br.votedThisStep = true
		case result := <-br.resultChannel:
			if result != nil {
				vote, err := br.addVoteInfo(result)
				if err != nil {
					// Log
					return
				}

				br.eventBus.Publish(msg.OutgoingAgreementTopic, vote)
			}

			br.incrementStep()
			br.reducing = false
		default:
			br.checkQueue()

			if !br.reducing {
				br.checkRoundChannel()
			}
		}
	}
}

func (br BlockReducer) checkQueue() {
	queuedMessages := br.queue.GetMessages(br.currentRound, br.currentStep)

	if queuedMessages != nil {
		for _, message := range queuedMessages {
			m := message.(*BlockReduction)
			br.process(m)
		}
	}
}

func (br *BlockReducer) checkRoundChannel() {
	select {
	case round := <-br.roundChannel:
		br.updateRound(round)
	default:
		break
	}
}

type voteCollector struct {
	voteChannel chan<- []byte
}

func (v *voteCollector) Collect(blockBuffer *bytes.Buffer) error {
	v.voteChannel <- blockBuffer.Bytes()
	return nil
}

// roundCollector is a simple wrapper over a channel to get round notifications
type roundCollector struct {
	roundChan chan<- uint64
}

// Collect as specified in the EventCollector interface. In this case Collect simply performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}
