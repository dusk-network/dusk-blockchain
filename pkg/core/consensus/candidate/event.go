package candidate

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// Candidate represents a block candidate that was generated for the consensus.
type Candidate struct {
	*block.Block
}

// Sender implements wire.Event.
// In this case, it returns nil, as the Candidate does not contain any field which
// denotes its sender.
func (c *Candidate) Sender() []byte {
	return nil
}

// Equal implements wire.Event.
// It checks if two Candidate events are the same.
func (c *Candidate) Equal(ev wire.Event) bool {
	other := ev.(*Candidate)
	return c.Header.Equals(other.Header)
}
