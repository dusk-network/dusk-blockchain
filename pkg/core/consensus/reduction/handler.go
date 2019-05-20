package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// ReductionHandler is responsible for performing operations that need to know
	// about specific event fields.
	reductionHandler struct {
		committee.Committee
		*UnMarshaller
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
		Committee:    committee,
		UnMarshaller: NewUnMarshaller(),
	}
}

func (b *reductionHandler) ExtractHeader(e wire.Event) *header.Header {
	ev := e.(*Reduction)
	return &header.Header{
		Round: ev.Round,
		Step:  ev.Step,
	}
}

func (b *reductionHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*Reduction)
	return encoding.Write256(r, ev.BlockHash)
}

// Verify the blockEvent
func (b *reductionHandler) Verify(e wire.Event) error {
	ev := e.(*Reduction)
	return msg.VerifyBLSSignature(ev.PubKeyBLS, ev.BlockHash, ev.SignedHash)
}

// Priority is not used for this handler
func (b *reductionHandler) Priority(ev1, ev2 wire.Event) bool {
	return true
}
