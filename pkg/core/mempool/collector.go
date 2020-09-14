package mempool

import (
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type intermediateBlockCollector struct {
	blkChan chan<- block.Block
}

func initIntermediateBlockCollector(sub eventbus.Subscriber) chan block.Block {
	blkChan := make(chan block.Block, 1)
	coll := &intermediateBlockCollector{blkChan}
	collectListener := eventbus.NewCallbackListener(coll.Collect)
	if config.Get().General.SafeCallbackListener {
		collectListener = eventbus.NewSafeCallbackListener(coll.Collect)
	}
	sub.Subscribe(topics.IntermediateBlock, collectListener)
	return blkChan
}

func (i *intermediateBlockCollector) Collect(blockMsg message.Message) {
	blk := blockMsg.Payload().(block.Block)
	i.blkChan <- blk
}
