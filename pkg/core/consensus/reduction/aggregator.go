package reduction

import "github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"

type aggregator struct {
	voteSets map[string]struct {
		*agreement.StepVotes
		sortedset.Set
	}
}
