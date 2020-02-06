package reduction

import "github.com/dusk-network/dusk-blockchain/pkg/core/consensus"

// Reducer abstracts the first and second step Reduction components
type Reducer interface {
	consensus.Component
	// Collect collects reduction consensus events in order to generate StepVotes or an Agreement
	Collect(consensus.InternalPacket) error
}
