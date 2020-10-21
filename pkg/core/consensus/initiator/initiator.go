package initiator

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// LaunchConsensus start the whole consensus algorithm
// TODO: managing the consensus should be performed within the chain, so to
// avoid excessive asynchronism. The chain should start and stop each round of
// the consensus explicitly
func LaunchConsensus(ctx context.Context, eventBroker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, w *wallet.Wallet, proxy transactions.Proxy) {
	// Setting up the consensus factory

	// settings := config.Get().Consensus
	// consensusTimeOut := time.Duration(settings.ConsensusTimeOut) * time.Second

	// f := factory.New(ctx, eventBroker, rpcBus, consensusTimeOut, &w.PublicKey, w.Keys(), proxy)
	// f.StartConsensus()

	// If we are on genesis, we should kickstart the consensus
	timeoutGetLastBlock := time.Duration(config.Get().Timeout.TimeoutGetLastBlock) * time.Second
	resp, err := rpcBus.Call(topics.GetLastBlock, rpcbus.NewRequest(bytes.Buffer{}), timeoutGetLastBlock)
	if err != nil {
		log.Panic(err)
	}
	blk := resp.(block.Block)

	if blk.Header.Height == 0 {
		msg := message.New(topics.Initialization, bytes.Buffer{})
		errList := eventBroker.Publish(topics.Initialization, msg)
		diagnostics.LogPublishErrors("initiator/initiator.go, LaunchConsensus, topics.Initialization", errList)
	}
}
