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
	// BlockEvent is a basic reduction event.
	BlockEvent struct {
		*consensus.EventHeader
		VotedHash  []byte
		SignedHash []byte
	}

	reductionEventUnmarshaller struct {
		*consensus.EventHeaderUnmarshaller
		*consensus.EventHeaderMarshaller
	}

	// BlockHandler is responsible for performing operations that need to know
	// about specific event fields.
	blockHandler struct {
		committee committee.Committee
		*reductionEventUnmarshaller
	}
)

// Equal as specified in the Event interface
func (e *BlockEvent) Equal(ev wire.Event) bool {
	other, ok := ev.(*BlockEvent)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

func newReductionEventUnmarshaller() *reductionEventUnmarshaller {
	return &reductionEventUnmarshaller{
		EventHeaderUnmarshaller: consensus.NewEventHeaderUnmarshaller(),
		EventHeaderMarshaller:   &consensus.EventHeaderMarshaller{},
	}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *reductionEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*BlockEvent)
	if err := a.EventHeaderUnmarshaller.Unmarshal(r, bev.EventHeader); err != nil {
		return err
	}

	if err := encoding.Read256(r, &bev.VotedHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &bev.SignedHash); err != nil {
		return err
	}

	return nil
}

func (a *reductionEventUnmarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*BlockEvent)
	if err := a.EventHeaderMarshaller.Marshal(r, bev.EventHeader); err != nil {
		return err
	}

	if err := encoding.Write256(r, bev.VotedHash); err != nil {
		return err
	}

	if err := encoding.Write256(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// newBlockHandler will return a BlockHandler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func newBlockHandler(committee committee.Committee) *blockHandler {
	return &blockHandler{
		committee:                  committee,
		reductionEventUnmarshaller: newReductionEventUnmarshaller(),
	}
}

func (b *blockHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*BlockEvent)
	h.Round = ev.Round
	h.Step = ev.Step
	h.PubKeyBLS = ev.PubKeyBLS
}

func (b *blockHandler) ExtractVoteHash(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*BlockEvent)
	if err := encoding.Write256(r, ev.VotedHash); err != nil {
		return err
	}
	return nil
}

// NewEvent returns a blockEvent
func (b *blockHandler) NewEvent() wire.Event {
	return &BlockEvent{}
}

// Verify the blockEvent
func (b *blockHandler) Verify(e wire.Event) error {
	ev := e.(*BlockEvent)

	if err := msg.VerifyBLSSignature(ev.PubKeyBLS, ev.VotedHash, ev.SignedHash); err != nil {
		return err
	}

	if !b.committee.IsMember(ev.PubKeyBLS) {
		return errors.New("block handler: voter not eligible to vote")
	}

	return nil
}
