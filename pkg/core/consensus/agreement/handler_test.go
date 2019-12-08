package agreement

import (
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/stretchr/testify/assert"
)

// TestMockAgreementEvent tests the general layout of a mock Agreement (i.e. the BitSet)
func TestMockAgreementEvent(t *testing.T) {
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 3, keys, p)
	assert.NotEqual(t, 0, ev.VotesPerStep[0].BitSet)
	assert.NotEqual(t, 0, ev.VotesPerStep[1].BitSet)
}

func TestVoteVerification(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(3)
	hash, _ := crypto.RandEntropy(32)
	ev := MockAgreementEvent(hash, 1, 3, keys, p)
	handler := newHandler(keys[0], *p)
	if !assert.NoError(t, handler.Verify(*ev)) {
		assert.FailNow(t, "problems in verification logic")
	}
}

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

// BenchmarkReconstructApk benchmarks the APK reconstruction.
// Note: the benchmark should be run with `gcflags="-l -N"` to prevent compiler
// optimization
func BenchmarkReconstructApk(b *testing.B) {

	sortedSet := sortedset.New()

	// creating a set of 100 random keys
	for i := 0; i < 100; i++ {
		keys, _ := key.NewRandConsensusKeys()
		sortedSet.Insert(keys.BLSPubKeyBytes)
	}

	b.ResetTimer()

	// benchmarking ReconstructApk
	for i := 0; i < b.N; i++ {
		_, _ = ReconstructApk(sortedSet)
	}

}
