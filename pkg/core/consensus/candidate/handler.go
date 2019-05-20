package candidate

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type candidateHandler struct {
}

func newCandidateHandler() *candidateHandler {
	return &candidateHandler{}
}

func (c *candidateHandler) NewEvent() wire.Event {
	return &CandidateEvent{block.NewBlock()}
}

func (c *candidateHandler) Marshal(r *bytes.Buffer, ev wire.Event) error {
	cev := ev.(*CandidateEvent)
	return cev.Encode(r)
}

func (c *candidateHandler) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	cev := ev.(*CandidateEvent)
	return cev.Decode(r)
}

func (c *candidateHandler) Verify(ev wire.Event) error {
	return nil // TODO: set up req-resp with chain
}

func (c *candidateHandler) ExtractHeader(ev wire.Event) *events.Header {
	cev := ev.(*CandidateEvent)
	return &events.Header{
		Round: cev.Header.Height,
	}
}
