// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestAccumulatorProcessing(t *testing.T) {
	nr := 10
	hlp := NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := NewHandler(hlp.Keys, *hlp.P)
	accumulator := newAccumulator(handler, 4)

	evs := hlp.Spawn(hash)
	for _, msg := range evs {
		accumulator.Process(msg)
	}

	accumulatedAggros := <-accumulator.CollectedVotesChan
	assert.Equal(t, 8, len(accumulatedAggros))
}
