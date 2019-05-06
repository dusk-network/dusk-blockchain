package agreement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestVoteVerification(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)

	ev := MockAggregatedAgreement(hash, 1, 1, 2)

	// mocking voters
	c := mockCommittee(2, true)

	handler := newHandler(c)

	assert.NoError(t, handler.Verify(ev))
}
