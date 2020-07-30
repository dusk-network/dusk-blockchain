package reduction

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// HaltMsg is used to communicate between reducers, and aggregators and timers.
type HaltMsg struct {
	Hash []byte
	Sv   []*message.StepVotes
}

// Reducer abstracts the first and second step Reduction components
type Reducer interface {
	consensus.Component
	// Collect collects reduction consensus events in order to generate StepVotes or an Agreement
	Collect(consensus.InternalPacket) error
}
