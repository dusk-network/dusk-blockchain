package reduction

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	sigSetEvent struct {
		*Event
		blockHash []byte
	}

	sigSetReductionUnmarshaller struct {
		*reductionEventUnmarshaller
	}

	// SigSetHandler is responsible for performing operations that need to know
	// about specific event fields.
	SigSetHandler struct {
		committee committee.Committee
		*sigSetReductionUnmarshaller
		blockHash []byte
	}
)

// Equal implements Event interface.
func (sse *sigSetEvent) Equal(e wire.Event) bool {
	return sse.Event.Equal(e) &&
		bytes.Equal(sse.blockHash, e.(*sigSetEvent).blockHash)
}

func newSigSetReductionUnmarshaller(validate func(*bytes.Buffer) error) *sigSetReductionUnmarshaller {
	return &sigSetReductionUnmarshaller{
		reductionEventUnmarshaller: newReductionEventUnmarshaller(validate),
	}
}

func (ssru *sigSetReductionUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetEvent := e.(*sigSetEvent)
	if err := ssru.reductionEventUnmarshaller.Unmarshal(r, sigSetEvent.Event); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetEvent.blockHash); err != nil {
		return err
	}

	return nil
}

func (ssru *sigSetReductionUnmarshaller) Marshal(r *bytes.Buffer, e wire.Event) error {
	sigSetEvent := e.(*sigSetEvent)
	if err := ssru.reductionEventUnmarshaller.Marshal(r, sigSetEvent.Event); err != nil {
		return err
	}

	if err := encoding.Write256(r, sigSetEvent.blockHash); err != nil {
		return err
	}

	return nil
}

// NewSigSetHandler will return a SigSetHandler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func NewSigSetHandler(eventBus *wire.EventBus, committee committee.Committee,
	validateFunc func(*bytes.Buffer) error) *SigSetHandler {

	phaseChannel := consensus.InitPhaseCollector(eventBus).BlockHashChan
	sigSetHandler := &SigSetHandler{
		committee:                   committee,
		sigSetReductionUnmarshaller: newSigSetReductionUnmarshaller(validateFunc),
	}

	go func() {
		for {
			sigSetHandler.blockHash = <-phaseChannel
		}
	}()
	return sigSetHandler
}

// NewEvent returns a sigSetEvent
func (s SigSetHandler) NewEvent() wire.Event {
	return &sigSetEvent{}
}

// Stage returns the round and step of the passed sigSetEvent
func (s SigSetHandler) Stage(e wire.Event) (uint64, uint8) {
	ev, ok := e.(*sigSetEvent)
	if !ok {
		return 0, 0
	}

	return ev.Round, ev.Step
}

// Hash returns the voted hash on the passed sigSetEvent
func (s SigSetHandler) Hash(e wire.Event) []byte {
	ev, ok := e.(*sigSetEvent)
	if !ok {
		return nil
	}

	return ev.VotedHash
}

// Verify the sigSetEvent
func (s SigSetHandler) Verify(e wire.Event) error {
	ev, ok := e.(*sigSetEvent)
	if !ok {
		return errors.New("block handler: type casting error")
	}

	if err := msg.VerifyBLSSignature(ev.PubKeyBLS, ev.VotedHash, ev.SignedHash); err != nil {
		return err
	}

	if !bytes.Equal(s.blockHash, ev.blockHash) {
		return errors.New("sig set handler: block hash mismatch")
	}

	if !s.committee.IsMember(ev.PubKeyBLS) {
		return errors.New("sig set handler: voter not eligible to vote")
	}

	return nil
}
