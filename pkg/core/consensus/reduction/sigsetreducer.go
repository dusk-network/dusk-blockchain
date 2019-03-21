package reduction

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	// SigSetReduction is a reduction event for the signature set phase.
	SigSetReduction struct {
		*Event
		winningBlockHash []byte
	}

	sigSetReductionUnmarshaller struct {
		*reductionEventUnmarshaller
	}

	// SigSetReducer is responsible for handling incoming signature set reduction
	// messages. It sends the proper messages to a reduction sequencer, and
	// processes the outcomes produced by this sequencer.
	SigSetReducer struct {
		*Reducer
		selectionChannel <-chan []byte
		winningBlockHash []byte
	}
)

// Equal implements Event interface.
func (ssr *SigSetReduction) Equal(e wire.Event) bool {
	return ssr.Event.Equal(e) &&
		bytes.Equal(ssr.winningBlockHash, e.(*SigSetReduction).winningBlockHash)
}

func newSigSetReductionUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return newReductionEventUnmarshaller(validate)
}

func (ssru *sigSetReductionUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetReduction := e.(*SigSetReduction)

	if err := ssru.reductionEventUnmarshaller.Unmarshal(r, sigSetReduction.Event); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetReduction.winningBlockHash); err != nil {
		return err
	}

	return nil
}

// NewSigSetReducer will create a new signature set reducer and return it. The
// signature set reducer will subscribe to the appropriate topics, and only needs
// to be started by calling Listen, after being returned.
func NewSigSetReducer(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committee committee.Committee, timeOut time.Duration) *SigSetReducer {

	selectionChannel := make(chan []byte, 1)

	setSelectionCollector := &selectionCollector{selectionChannel}
	wire.NewEventSubscriber(eventBus, setSelectionCollector,
		string(msg.SelectionResultTopic)).Accept()

	sigSetReducer := &SigSetReducer{
		Reducer:          newReducer(eventBus, validateFunc, committee, timeOut),
		selectionChannel: selectionChannel,
	}

	wire.NewEventSubscriber(eventBus, sigSetReducer,
		string(topics.SigSetReduction)).Accept()

	return sigSetReducer
}

// Listen will start the signature set reducer. It will wait for messages on it's
// event channels, and handle them accordingly.
func (sr *SigSetReducer) Listen() {
	for {
		select {
		case round := <-sr.roundChannel:
			sr.stopChannel <- true
			sr.winningBlockHash = nil
			sr.updateRound(round)
		case blockHash := <-sr.phaseUpdateChannel:
			sr.winningBlockHash = blockHash
		case setHash := <-sr.selectionChannel:
			if sr.winningBlockHash != nil {
				go sr.startSequencer(setHash)
				go sr.listenSequencer()
			}
		}
	}
}

// Collect implements the EventCollector interface.
// Unmarshal a signature set reduction message, verify the signature and then
// pass it down for processing.
func (sr *SigSetReducer) Collect(buffer *bytes.Buffer) error {
	event := &SigSetReduction{}
	if err := sr.unmarshaller.Unmarshal(buffer, event); err != nil {
		return err
	}

	// verify correctness of included BLS public key and signature
	if err := msg.VerifyBLSSignature(event.PubKeyBLS, event.VotedHash,
		event.SignedHash); err != nil {

		return errors.New("sig set reduction: BLS verification failed")
	}

	if !bytes.Equal(sr.winningBlockHash, event.winningBlockHash) {
		return errors.New("sig set reduction: vote has wrong winning block hash")
	}

	sr.process(event.Event)
	return nil
}
