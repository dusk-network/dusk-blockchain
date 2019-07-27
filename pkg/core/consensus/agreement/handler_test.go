package agreement

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

func TestVoteVerification(t *testing.T) {
	// mocking voters
	c, keys := MockCommittee(2, true, 2)
	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 1, keys)
	handler := newHandler(c, user.Keys{})
	assert.NoError(t, handler.Verify(ev))
}
