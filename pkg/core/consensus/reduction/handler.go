package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// ReductionHandler is responsible for performing operations that need to know
	// about specific event fields.
	reductionHandler struct {
		committee.Committee
		*events.ReductionUnMarshaller
	}

	handler interface {
		consensus.AccumulatorHandler
		MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error
	}
)

// newReductionHandler will return a ReductionHandler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func newReductionHandler(committee committee.Committee) *reductionHandler {
	return &reductionHandler{
		Committee:             committee,
		ReductionUnMarshaller: events.NewReductionUnMarshaller(),
	}
}

func (b *reductionHandler) ExtractHeader(e wire.Event, h *events.Header) {
	ev := e.(*events.Reduction)
	h.Round = ev.Round
	h.Step = ev.Step
}

func (b *reductionHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*events.Reduction)
	return encoding.Write256(r, ev.VotedHash)
}

// NewEvent returns a blockEvent
func (b *reductionHandler) NewEvent() wire.Event {
	return events.NewReduction()
}

// Verify the blockEvent
func (b *reductionHandler) Verify(e wire.Event) error {
	ev := e.(*events.Reduction)
	return msg.VerifyBLSSignature(ev.PubKeyBLS, ev.VotedHash, ev.SignedHash)
}

// Priority is not used for this handler
func (b *reductionHandler) Priority(ev1, ev2 wire.Event) wire.Event {
	return nil
}
