package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// ReductionHandler is responsible for performing operations that need to know
	// about specific event fields.
	reductionHandler struct {
		user.Keys
		Reducers
		*UnMarshaller
	}
)

// newReductionHandler will return a ReductionHandler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func newReductionHandler(committee Reducers, keys user.Keys) *reductionHandler {
	return &reductionHandler{
		Keys:         keys,
		Reducers:     committee,
		UnMarshaller: NewUnMarshaller(),
	}
}

// AmMember checks if we are part of the committee.
func (b *reductionHandler) AmMember(round uint64, step uint8) bool {
	return b.Reducers.IsMember(b.Keys.BLSPubKeyBytes, round, step)
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

// Verify the BLS signature of the Reduction event.
func (b *reductionHandler) Verify(e wire.Event) error {
	ev := e.(*Reduction)
	info := new(bytes.Buffer)
	if err := header.MarshalSignableVote(info, ev.Header); err != nil {
		return err
	}
	return msg.VerifyBLSSignature(ev.PubKeyBLS, info.Bytes(), ev.SignedHash)
}
