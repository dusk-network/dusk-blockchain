package reduction

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type (
	// BlockReducer is responsible for handling incoming block reduction messages.
	// It sends the proper messages to a reduction sequencer, and processes the
	// outcomes produced by this sequencer.
	BlockReducer struct {
		*Reducer
		selectionChannel <-chan []byte
		sequencerDone    bool
	}

	selectionCollector struct {
		selectionChannel chan<- []byte
	}
)

// NewBlockReducer will create a new block reducer and return it. The block reducer
// will subscribe to the appropriate topics, and only needs to be started by
// calling Listen, after being returned.
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
			br.sequencerDone = false
			br.updateRound(round)
		case <-br.phaseUpdateChannel:
			br.stopChannel <- true
			br.sequencerDone = true
		case blockHash := <-br.selectionChannel:
			if !br.sequencerDone {
				br.vote(blockHash, msg.OutgoingReductionTopic)
				go br.startSequencer()
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
