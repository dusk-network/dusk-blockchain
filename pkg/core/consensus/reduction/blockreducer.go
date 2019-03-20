package reduction

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type (
	// BlockReduction is a block phase reduction event.
	BlockReduction = Event

	// BlockReducer is responsible for handling incoming block reduction messages.
	// It sends the proper messages to a reduction selector, and processes the
	// outcomes produced by this selector.
	BlockReducer struct {
		eventBus *wire.EventBus
		*reductionSequencer
		unmarshaller wire.EventUnmarshaller

		// channels linked to subscribers
		blockSelectionChannel <-chan []byte
		roundChannel          <-chan uint64
		phaseUpdateChannel    <-chan []byte

		stopChannel      chan bool
		sequencerStopped bool
	}

	blockSelectionCollector struct {
		blockSelectionChannel chan<- []byte
	}
)

// NewBlockReducer will create a new block reducer and return it. The block reducer
// will subscribe to the appropriate topics, and only needs to be started by calling
// Listen, after being returned.
func NewBlockReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timeOut time.Duration) *BlockReducer {

	blockSelectionChannel := make(chan []byte, 1)

	reductionSequencer := newReductionSequencer(committee, timeOut)

	blockSelectionCollector := &blockSelectionCollector{blockSelectionChannel}
	wire.NewEventSubscriber(eventBus, blockSelectionCollector,
		string(msg.SelectionResultTopic)).Accept()

	roundCollector := consensus.InitRoundCollector(eventBus)
	phaseUpdateCollector := consensus.InitPhaseCollector(eventBus)

	blockReducer := &BlockReducer{
		eventBus:              eventBus,
		reductionSequencer:    reductionSequencer,
		unmarshaller:          newReductionEventUnmarshaller(validateFunc),
		blockSelectionChannel: blockSelectionChannel,
		roundChannel:          roundCollector.RoundChan,
		phaseUpdateChannel:    phaseUpdateCollector.BlockHashChan,
	}

	wire.NewEventSubscriber(eventBus, blockReducer, string(topics.BlockReduction)).Accept()

	return blockReducer
}

// Listen will start the block reducer. It will wait for messages on it's event
// channels, and handle them accordingly.
func (br *BlockReducer) Listen() {
	for {
		select {
		case round := <-br.roundChannel:
			br.updateRound(round)
		case <-br.phaseUpdateChannel:
			br.stopChannel <- true
			br.sequencerStopped = true
		case blockHash := <-br.blockSelectionChannel:
			if !br.sequencerStopped {
				go br.startSequencer(blockHash)
				go br.listenSequencer()
			}
		}
	}
}

// listen for output from the sequencer, and handle accordingly
func (br *BlockReducer) listenSequencer() {
	for {
		select {
		case <-br.stopChannel:
			// stop listening to the sequencer
			return
		case hash := <-br.outgoingReductionChannel:
			voteData, err := br.addVoteInfo(hash)
			if err != nil {
				// Log
				return
			}

			br.eventBus.Publish(msg.OutgoingReductionTopic, voteData)
		case hashAndVoteSet := <-br.outgoingAgreementChannel:
			voteData, err := br.addVoteInfo(hashAndVoteSet)
			if err != nil {
				// Log
				return
			}

			br.eventBus.Publish(msg.OutgoingAgreementTopic, voteData)

			// sequencer is now done, so we can return
			return
		}
	}
}

// Collect implements the EventCollector interface.
// Unmarshal a block reduction message, verify the signature and then
// pass it down for processing.
func (br *BlockReducer) Collect(buffer *bytes.Buffer) error {
	event := &Event{}
	if err := br.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	// verify correctness of included BLS public key and signature
	if err := msg.VerifyBLSSignature(event.PubKeyBLS, event.VotedHash,
		event.SignedHash); err != nil {

		return errors.New("block reduction: BLS verification failed")
	}

	br.process(event)
	return nil
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

// Collect implements the EventCollector interface.
// Will simply send the received buffer as a slice of bytes.
func (v *blockSelectionCollector) Collect(blockBuffer *bytes.Buffer) error {
	v.blockSelectionChannel <- blockBuffer.Bytes()
	return nil
}
