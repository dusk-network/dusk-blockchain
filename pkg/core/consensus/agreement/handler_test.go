package agreement

import (
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestMockAgreementEvent(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(50)
	vc := p.CreateVotingCommittee(1, 1, 50)
	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 1, keys, vc)
	fmt.Println(ev.VotesPerStep[0].BitSet)
	fmt.Println(ev.VotesPerStep[1].BitSet)

}

func TestVoteVerification(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(50)
	vc := p.CreateVotingCommittee(1, 1, 50)
	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 1, keys, vc)
	handler := newHandler(keys[0], *p)
	assert.NoError(t, handler.Verify(*ev))
}
