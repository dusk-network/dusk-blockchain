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
	// SigSetCollector is the public collector used outside of the Broker (which use the unexported one)
	SigSetCollector struct{}

	// SigSetEvent is the event related to the completed reduction of a Signature Set for a specific round (TODO: and step?)
	SigSetEvent struct {
		*BlockEvent
		blockHash []byte
	}

	sigSetUnmarshaller struct {
		*blockUnMarshaller
	}

	// sigSetHandler is responsible for performing operations that need to know
	// about specific event fields.
	sigSetHandler struct {
		committee committee.Committee
		*sigSetUnmarshaller
		blockHash []byte
	}
)

// Equal implements Event interface.
func (sse *SigSetEvent) Equal(e wire.Event) bool {
	return sse.BlockEvent.Equal(e) &&
		bytes.Equal(sse.blockHash, e.(*SigSetEvent).blockHash)
}

func newSigSetUnMarshaller() *sigSetUnmarshaller {
	return &sigSetUnmarshaller{
		unMarshaller: newUnMarshaller(),
	}
}

func (ssru *sigSetUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetEvent := e.(*SigSetEvent)
	if err := ssru.blockUnMarshaller.Unmarshal(r, sigSetEvent.BlockEvent); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetEvent.blockHash); err != nil {
		return err
	}

	return nil
}

func (ssru *sigSetUnmarshaller) Marshal(r *bytes.Buffer, e wire.Event) error {
	sigSetEvent := e.(*SigSetEvent)
	if err := ssru.blockUnMarshaller.Marshal(r, sigSetEvent.BlockEvent); err != nil {
		return err
	}

	if err := encoding.Write256(r, sigSetEvent.blockHash); err != nil {
		return err
	}

	return nil
}

// newSigSetHandler will return a SigSetHandler, injected with the passed committee and an unmarshaller
func newSigSetHandler(eventBus *wire.EventBus, committee committee.Committee) *sigSetHandler {
	phaseChannel := consensus.InitPhaseUpdate(eventBus)
	sigSetHandler := &sigSetHandler{
		committee:                   committee,
		sigSetReductionUnmarshaller: newSigSetReductionUnmarshaller(),
	}

	go func() {
		for {
			sigSetHandler.blockHash = <-phaseChannel
		}
	}()
	return sigSetHandler
}

// NewEvent returns a sigSetEvent
func (s sigSetHandler) NewEvent() wire.Event {
	return &SigSetEvent{}
}

func (b *sigSetHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*BlockEvent)
	h.Round = ev.Round
	h.Step = ev.Step
	h.PubKeyBLS = ev.PubKeyBLS
}

func (b *sigSetHandler) ExtractVoteHash(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*BlockEvent)
	if err := encoding.Write256(r, ev.VotedHash); err != nil {
		return err
	}
	return nil
}

// Verify the sigSetEvent
func (s sigSetHandler) Verify(e wire.Event) error {
	ev := e.(*SigSetEvent)
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
