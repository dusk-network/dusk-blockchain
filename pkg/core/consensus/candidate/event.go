package candidate

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type CandidateEvent struct {
	*block.Block
}

func (c *CandidateEvent) Sender() []byte {
	return nil
}

func (c *CandidateEvent) Equal(ev wire.Event) bool {
	other := ev.(*CandidateEvent)
	return c.Header.Equals(other.Header)
}
