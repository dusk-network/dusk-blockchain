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
		h := committee.NewHandler(ks[0], *p)
		h.AmMember(1, 0, 10)
	})
}
