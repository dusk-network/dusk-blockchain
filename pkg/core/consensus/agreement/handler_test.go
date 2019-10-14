package agreement

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
)

func TestVoteVerification(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(50)

	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 50))
	handler := newHandler(key.ConsensusKeys{})
	handler.Handler.Provisioners = *p
	assert.NoError(t, handler.Verify(ev))
}

func TestSignEd25519(t *testing.T) {
	k, _ := user.NewRandKeys()
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	buf := MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 50))

	handler := newHandler(k)
	signed := handler.signEd25519(buf.Bytes())

	signature := make([]byte, 64)
	assert.NoError(t, encoding.Read512(signed, signature))
	assert.True(t, ed25519.Verify(*k.EdPubKey, buf.Bytes(), signature))
}
