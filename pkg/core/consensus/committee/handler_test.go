// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package committee_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/stretchr/testify/assert"
)

// Test that checking the committee on step 0 does not
// cause a panic.
func TestStep0Committee(t *testing.T) {
	assert.NotPanics(t, func() {
		p, ks := consensus.MockProvisioners(10)
		h := committee.NewHandler(ks[0], *p, []byte{0, 0, 0, 0})
		h.AmMember(1, 0, 10)
	})
}
