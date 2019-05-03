package agreement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestVoteVerification(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	keys, ev1 := MockVote(nil, hash, 1, 1)
	_, ev2 := MockVote(keys, hash, 1, 2)

	agreement := MockAgreement(keys, hash, 1, 2, []wire.Event{ev1, ev2})

	// mocking voters
	c := mockCommittee(2, true)

	handler := newHandler(c)

	assert.NoError(t, handler.Verify(agreement))
}
