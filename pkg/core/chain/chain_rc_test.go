// +build race

package chain

import (
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// Tests below can be performed only with race detector enabled

func TestConcurrentBlock(t *testing.T) {

	// Set up a chain instance with mocking verifiers
	startingHeight := uint64(1)
	eb, rpc, chain := setupChainTest(startingHeight, true)
	go chain.Listen()

	// Set up a goroutine to provide a mock topics.RoundUpdates
	go mockRoundResults(chain, eb)

	// Set up a goroutine to provide a mock topics.GetCandidate
	go mockGetCandidate(rpc)

	// Disable counter.IsSyncing check at the beginning of block accepting
	// by providing empty callback
	chain.onBeginAccepting = func() bool { return true }

	// Switch to syncing mode so that requestRoundResults will be called
	chain.counter.StartSyncing(1)

	time.Sleep(1 * time.Second)
	var wg sync.WaitGroup
	for n := 0; n < 50; n++ {

		wg.Add(2)

		// Publish concurrently topics.Block from different goroutines
		go func() {
			defer wg.Done()

			blk := helper.RandomBlock(1, 3)
			msg := message.New(topics.Block, *blk)
			eb.Publish(topics.Block, msg)
		}()

		// Publish concurrently topics.Certificate from different goroutines
		go func() {
			defer wg.Done()
			mockCertificate(eb)
		}()
	}

	wg.Wait()
	time.Sleep(2 * time.Second)
}

//nolint:unused
func mockCertificate(eb *eventbus.EventBus) {
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	ag := message.MockAgreement(hash, 2, 7, keys, p, 10)

	cert := message.NewCertificate(ag, nil)
	msg := message.New(topics.Certificate, cert)
	eb.Publish(topics.Certificate, msg)
}

//nolint:unused
func mockRoundResults(c *Chain, eb *eventbus.EventBus) {

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	genesis := config.DecodeGenesis()

	for {
		// Read Chain request for RoundResults
		streamer.Read()

		// Provide RoundResults directly into eventbus
		cert := block.EmptyCertificate()
		cm := message.MakeCandidate(genesis, cert)

		msg := message.New(topics.RoundResults, cm)
		eb.Publish(topics.RoundResults, msg)
	}

}

//nolint:unused
func mockGetCandidate(rpc *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 1)
	rpc.Register(topics.GetCandidate, c)

	blk := helper.RandomBlock(1, 1)
	cert := block.EmptyCertificate()
	cm := message.MakeCandidate(blk, cert)

	for {
		r := <-c
		r.RespChan <- rpcbus.NewResponse(cm, nil)
	}
}
