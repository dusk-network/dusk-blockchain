package candidate

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type candidateHandler struct {
}

func newCandidateHandler() *candidateHandler {
	return &candidateHandler{}
}

func (c *candidateHandler) Marshal(r *bytes.Buffer, ev wire.Event) error {
	cev := ev.(*CandidateEvent)
	return cev.Encode(r)
}

func (c *candidateHandler) Deserialize(r *bytes.Buffer) (wire.Event, error) {
	cev := &CandidateEvent{}
	if err := cev.Decode(r); err != nil {
		return nil, err
	}

	return cev, nil
}

func (c *candidateHandler) Verify(ev wire.Event) error {
	return nil // TODO: set up req-resp with chain
}

func (c *candidateHandler) ExtractHeader(ev wire.Event) *header.Header {
	cev := ev.(*CandidateEvent)
	return &header.Header{
		Round: cev.Header.Height,
	}
}
