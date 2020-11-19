package selection

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

func TestPriority(t *testing.T) {

	// mock candidate
	genesis := config.DecodeGenesis()
	c := message.MakeCandidate(genesis)

	hdr := header.Mock()
	ev := message.MockScore(hdr, c)

	// Comparing the same event should return true
	handler := NewScoreHandler(nil)
	assert.True(t, handler.Priority(ev, ev))
}
