package reduction

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LaunchBlockReducer creates and wires a broker, initiating the components that
// have to do with Block Reduction
func LaunchBlockReducer(eventBus *wire.EventBus, committee committee.Committee,
	timeout time.Duration) *broker {

	scoreChan := selection.InitBestScoreUpdate(eventBus)
	handler := newBlockHandler(committee)
	broker := newBroker(eventBus, handler, scoreChan, committee,
		topics.BlockReduction, string(msg.OutgoingBlockReductionTopic),
		string(msg.OutgoingBlockAgreementTopic),
		string(msg.BlockGenerationTopic), timeout)

	go broker.Listen()
	return broker
}

type (
	// BlockHandler is responsible for performing operations that need to know
	// about specific event fields.
	blockHandler struct {
		committee committee.Committee
		*unMarshaller
	}
)

// newBlockHandler will return a BlockHandler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func newBlockHandler(committee committee.Committee) *blockHandler {
	return &blockHandler{
		committee:    committee,
		unMarshaller: newUnMarshaller(msg.VerifyEd25519Signature),
	}
}

func (b *blockHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*committee.ReductionEvent)
	h.Round = ev.Round
	h.Step = ev.Step
}

func (b *blockHandler) EmbedVoteHash(e wire.Event, r *bytes.Buffer) error {
	var votedHash []byte
	if e == nil {
		votedHash = make([]byte, 32)
	} else {
		ev := e.(*committee.ReductionEvent)
		votedHash = ev.VotedHash
	}
	if err := encoding.Write256(r, votedHash); err != nil {
		return err
	}
	return nil
}

// NewEvent returns a blockEvent
func (b *blockHandler) NewEvent() wire.Event {
	return &committee.ReductionEvent{
		EventHeader: &consensus.EventHeader{},
	}
}

// Verify the blockEvent
func (b *blockHandler) Verify(e wire.Event) error {
	ev := e.(*committee.ReductionEvent)
	if !b.committee.IsMember(ev.PubKeyBLS) {
		return errors.New("block handler: voter not eligible to vote")
	}

	if err := msg.VerifyBLSSignature(ev.PubKeyBLS, ev.VotedHash, ev.SignedHash); err != nil {
		return err
	}

	return nil
}

// Priority is not used for this handler
func (b *blockHandler) Priority(ev1, ev2 wire.Event) wire.Event {
	return nil
}
