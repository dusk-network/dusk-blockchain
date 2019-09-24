package agreement

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestVoteVerification(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(50)

	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 50))
	handler := newHandler(user.Keys{})
	handler.Handler.Provisioners = *p
	assert.NoError(t, handler.Verify(ev))
}
