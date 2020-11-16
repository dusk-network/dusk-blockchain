// +build race

package chain

import (
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// Tests below can be performed only with race detector enabled
func TestConcurrentBlock(t *testing.T) {
	// Set up a chain instance with mocking verifiers
	startingHeight := uint64(1)
	_, chain := setupChainTest(t, startingHeight)

	BLSKeys, _ := key.NewRandKeys()
	pk := keys.PublicKey{
		AG: &common.JubJubCompressed{Data: make([]byte, 32)},
		BG: &common.JubJubCompressed{Data: make([]byte, 32)},
	}

	go chain.SetupConsensus(pk, BLSKeys)

	var wg sync.WaitGroup
	for n := 0; n < 50; n++ {
		wg.Add(2)

		// Publish concurrently topics.Block from different goroutines
		go func() {
			defer wg.Done()

			blk := helper.RandomBlock(1, 3)
			msg := message.New(topics.Block, *blk)
			chain.ProcessBlock(msg)
		}()

		// Publish concurrently topics.Certificate from different goroutines
		go func() {
			defer wg.Done()
			chain.handleCertificateMessage(mockCertificate())
		}()
	}

	wg.Wait()
	time.Sleep(2 * time.Second)
}

//nolint:unused
func mockCertificate() (*block.Certificate, []byte) {
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	ag := message.MockAgreement(hash, 2, 7, keys, p, 10)
	return ag.GenerateCertificate(), hash
}
