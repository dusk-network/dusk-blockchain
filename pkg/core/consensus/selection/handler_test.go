package selection

import (
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestPriority(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	ev := MockSelectionEvent(hash)

	// Comparing the same event should return true
	handler := NewScoreHandler(nil)
	assert.True(t, handler.Priority(*ev, *ev))
}
