package initiator

import (
	"bytes"
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// LaunchConsensus start the whole consensus algorithm
func LaunchConsensus(ctx context.Context, eventBroker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, w *wallet.Wallet, _ *chainsync.Counter, proxy *transactions.Proxy) {
	// Setting up the consensus factory
	f := factory.New(ctx, eventBroker, rpcBus, config.ConsensusTimeOut, &w.SecretKey, &w.PublicKey, w.Keys(), proxy)
	f.StartConsensus()

	// If we are on genesis, we should kickstart the consensus
	resp, err := rpcBus.Call(topics.GetLastBlock, rpcbus.NewRequest(bytes.Buffer{}), 0)
	if err != nil {
		log.Panic(err)
	}
	blk := resp.(block.Block)

	if blk.Header.Height == 0 {
		msg := message.New(topics.Initialization, bytes.Buffer{})
		eventBroker.Publish(topics.Initialization, msg)
	}
}
