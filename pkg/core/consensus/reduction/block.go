package reduction

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LaunchBlockReducer creates and wires a broker, initiating the components that have to do with Block Reduction
func LaunchBlockReducer(eventBus *wire.EventBus, committee committee.Committee, timeout time.Duration) *broker {
	handler := newBlockHandler(committee)
	broker := newBroker(eventBus, handler, committee, string(msg.BlockSelectionTopic), string(topics.BlockReduction), string(msg.OutgoingBlockReductionTopic), string(msg.OutgoingBlockAgreementTopic), timeout)
	go broker.Listen()
	return broker
}

type (
	// BlockEvent is a basic reduction event.
	BlockEvent struct {
		*consensus.EventHeader
		VotedHash  []byte
		SignedHash []byte
	}

	blockUnMarshaller struct {
		*unMarshaller
	}

	// BlockHandler is responsible for performing operations that need to know
	// about specific event fields.
	blockHandler struct {
		committee committee.Committee
		*blockUnMarshaller
	}
)

// Equal as specified in the Event interface
func (e *BlockEvent) Equal(ev wire.Event) bool {
	other, ok := ev.(*BlockEvent)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

func newBlockUnMarshaller() *blockUnMarshaller {
	return &blockUnMarshaller{
		unMarshaller: newUnMarshaller(),
	}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *blockUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
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

// Marshal solely the VotedHash and SignedHash. Marshalling of the EventHeader is done separately since the round and step are managed elsewhere
func (a *blockUnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*BlockEvent)
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
		committee:         committee,
		blockUnMarshaller: newBlockUnMarshaller(),
	}
}

func (b *blockHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*BlockEvent)
	h.Round = ev.Round
	h.Step = ev.Step
}

func (b *blockHandler) EmbedVoteHash(e wire.Event, r *bytes.Buffer) error {
	var votedHash []byte
	if e == nil {
		votedHash = make([]byte, 32)
	} else {
		ev := e.(*BlockEvent)
		votedHash = ev.VotedHash
	}
	if err := encoding.Write256(r, votedHash); err != nil {
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

// Priority is not used for this handler
func (b *blockHandler) Priority(ev1, ev2 wire.Event) wire.Event {
	return nil
}
