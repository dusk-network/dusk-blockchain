package reduction

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type (
	// BlockReduction is a block phase reduction event.
	BlockReduction = Event

	// Reducer is a collection of common fields and methods, both used by the
	// BlockReducer and SigSetReducer.
	Reducer struct {
		eventBus *wire.EventBus
		*reductionSequencer
		unmarshaller wire.EventUnmarshaller

		// channels linked to subscribers
		roundChannel       <-chan uint64
		phaseUpdateChannel <-chan []byte

		stopChannel chan bool
	}

	// BlockReducer is responsible for handling incoming block reduction messages.
	// It sends the proper messages to a reduction sequencer, and processes the
	// outcomes produced by this sequencer.
	BlockReducer struct {
		*Reducer
		selectionChannel <-chan []byte
	}

	selectionCollector struct {
		selectionChannel chan<- []byte
	}
)

// newReducer will create a new standard reducer and return it.
func newReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timeOut time.Duration) *Reducer {

	reductionSequencer := newReductionSequencer(committee, timeOut)

	roundCollector := consensus.InitRoundCollector(eventBus)
	phaseUpdateCollector := consensus.InitPhaseCollector(eventBus)

	return &Reducer{
		eventBus:           eventBus,
		reductionSequencer: reductionSequencer,
		unmarshaller:       newReductionEventUnmarshaller(validateFunc),
		roundChannel:       roundCollector.RoundChan,
		phaseUpdateChannel: phaseUpdateCollector.BlockHashChan,
	}
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

// listen for output from the sequencer, and handle accordingly
func (r *Reducer) listenSequencer() {
	for {
		select {
		case <-r.stopChannel:
			// stop listening to the sequencer
			return
		case hash := <-r.outgoingReductionChannel:
			voteData, err := r.addVoteInfo(hash)
			if err != nil {
				// Log
				return
			}

			r.eventBus.Publish(msg.OutgoingReductionTopic, voteData)
		case hashAndVoteSet := <-r.outgoingAgreementChannel:
			voteData, err := r.addVoteInfo(hashAndVoteSet)
			if err != nil {
				// Log
				return
			}

			r.eventBus.Publish(msg.OutgoingAgreementTopic, voteData)

			// sequencer is now done, so we can return
			return
		}
	}
}

func NewBlockReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timeOut time.Duration) *BlockReducer {

	selectionChannel := make(chan []byte, 1)
	blockSelectionCollector := &selectionCollector{selectionChannel}
	wire.NewEventSubscriber(eventBus, blockSelectionCollector,
		string(msg.SelectionResultTopic)).Accept()

	blockReducer := &BlockReducer{
		Reducer:          newReducer(eventBus, validateFunc, committee, timeOut),
		selectionChannel: selectionChannel,
	}

	wire.NewEventSubscriber(eventBus, blockReducer,
		string(topics.BlockReduction)).Accept()

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
			br.sequencerDone = true
		case blockHash := <-br.selectionChannel:
			if !br.sequencerDone {
				go br.startSequencer(blockHash)
				go br.listenSequencer()
			}
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

// Collect implements the EventCollector interface.
// Will simply send the received buffer as a slice of bytes.
func (s *selectionCollector) Collect(buffer *bytes.Buffer) error {
	s.selectionChannel <- buffer.Bytes()
	return nil
}
