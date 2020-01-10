package mempool

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
)

type intermediateBlockCollector struct {
	blkChan chan<- block.Block
}

func initIntermediateBlockCollector(sub eventbus.Subscriber) chan block.Block {
	blkChan := make(chan block.Block, 1)
	coll := &intermediateBlockCollector{blkChan}
	l := eventbus.NewCallbackListener(coll.Collect)
	sub.Subscribe(topics.IntermediateBlock, l)
	return blkChan
}

func (i *intermediateBlockCollector) Collect(m bytes.Buffer) error {
	blk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&m, blk); err != nil {
		return err
	}

	i.blkChan <- *blk
	return nil
}
