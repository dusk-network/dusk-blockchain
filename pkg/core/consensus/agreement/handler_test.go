package agreement

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// TestMockAgreementEvent tests the general layout of a mock Agreement (i.e. the BitSet)
func TestMockAgreementEvent(t *testing.T) {
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, 1, 3, keys, p)
	assert.NotEqual(t, 0, ev.VotesPerStep[0].BitSet)
	assert.NotEqual(t, 0, ev.VotesPerStep[1].BitSet)
}

func TestVoteVerification(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(3)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, 1, 3, keys, p)
	handler := NewHandler(keys[0], *p)
	if !assert.NoError(t, handler.Verify(ev)) {
		assert.FailNow(t, "problems in verification logic")
	}
}

/*
func TestConsensusEventVerification(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	hash, _ := crypto.RandEntropy(32)
	for i := 0; i < 3; i++ {
		ce := MockConsensusEvent(hash, 1, 3, keys, p, i)
		ev, err := convertToAgreement(ce)
		assert.NoError(t, err)
		handler := newHandler(keys[0], *p)
		if !assert.NoError(t, handler.Verify(*ev)) {
			assert.FailNow(t, fmt.Sprintf("error at %d iteration", 0))
		}
	}
}
*/
