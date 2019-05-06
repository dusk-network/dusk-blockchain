package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

type reductionSigner struct {
	*eventSigner
	*events.OutgoingReductionUnmarshaller
}

func NewReductionSigner(keys *user.Keys) *reductionSigner {
	return &reductionSigner{
		eventSigner:                   newEventSigner(keys),
		OutgoingReductionUnmarshaller: events.NewOutgoingReductionUnmarshaller(),
	}
}

func (bs *reductionSigner) Sign(buf *bytes.Buffer) error {
	return events.SignReduction(buf, bs.Keys)
}
