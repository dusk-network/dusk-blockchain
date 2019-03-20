package reduction

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type (

	// BlockReduction is a block phase reduction event.
	BlockReduction = Event

	// blockReductionCollector is responsible for unmarshalling incoming
	// block reduction messages, and then passing them to the BlockReducer
	// to be processed accordingly.
	blockReductionCollector struct {
		*reductionCollector
		unmarshaller wire.EventUnmarshaller
	}

	// BlockReducer is responsible for handling incoming block reduction messages.
	// It sends the proper messages to a reduction selector, and processes the
	// outcomes produced by this selector.
	BlockReducer struct {
		eventBus            *wire.EventBus
		reductionSubscriber *wire.EventSubscriber
		voteSubscriber      *wire.EventSubscriber
		roundSubscriber     *wire.EventSubscriber
		*blockReductionCollector

		// channels linked to subscribers
		voteChannel  <-chan []byte
		roundChannel <-chan uint64
	}

	// roundCollector is a simple wrapper over a channel to get round notifications
	roundCollector struct {
		roundChan chan<- uint64
	}

	voteCollector struct {
		voteChannel chan<- []byte
	}
)

func newBlockReductionCollector(committee committee.Committee, timeOut time.Duration,
	validateFunc func(*bytes.Buffer) error) *blockReductionCollector {

	reductionCollector := newReductionCollector(committee, timeOut)

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
		brc.disconnectSelector()
		brc.reducing = false
	}

	brc.Clear()
	brc.getQueuedMessages()
}

func (brc *blockReductionCollector) Collect(buffer *bytes.Buffer) error {
	event := &Event{}
	if err := brc.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	brc.process(event)
	return nil
}

func NewBlockReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timeOut time.Duration) *BlockReducer {

	voteChannel := make(chan []byte, 1)
	roundChannel := make(chan uint64, 1)

	reductionCollector := newBlockReductionCollector(committee, timeOut,
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

func (br *BlockReducer) Listen() {
	go br.reductionSubscriber.Accept()
	go br.voteSubscriber.Accept()
	go br.roundSubscriber.Accept()

	for {
		select {
		case round := <-br.roundChannel:
			br.updateRound(round)
		case blockHash := <-br.voteChannel:
			if !br.votedThisStep {
				if err := br.sendVoteData(blockHash, msg.OutgoingReductionTopic); err != nil {
					// Log
					return
				}

				br.votedThisStep = true
			}
		case blockHash := <-br.hashChannel:
			br.incrementStep()
			if err := br.sendVoteData(blockHash, msg.OutgoingReductionTopic); err != nil {
				// Log
				return
			}

			br.votedThisStep = true
		case result := <-br.resultChannel:
			if result != nil {
				if err := br.sendVoteData(result, msg.OutgoingAgreementTopic); err != nil {
					// Log
					return
				}
			}

			br.reducing = false
			br.incrementStep()
		}
	}
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

func (br BlockReducer) sendVoteData(data []byte, topic string) error {
	voteData, err := br.addVoteInfo(data)
	if err != nil {
		return err
	}

	br.eventBus.Publish(topic, voteData)
	return nil
}

// Collect implements the EventCollector interface.
// Will simply send the received buffer as a slice of bytes.
func (v *voteCollector) Collect(blockBuffer *bytes.Buffer) error {
	v.voteChannel <- blockBuffer.Bytes()
	return nil
}

// Collect implements the EventCollector interface.
// In this case Collect simply performs unmarshalling of the round event.
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}
