package selection

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestPriority(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	hdr := header.Mock()
	ev := message.MockScore(hdr, hash)

	// Comparing the same event should return true
	handler := NewScoreHandler(nil)
	assert.True(t, handler.Priority(ev, ev))
}
