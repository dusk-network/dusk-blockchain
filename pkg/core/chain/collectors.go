package chain

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	blockCollector struct {
		blockChannel chan<- *block.Block
	}
)

func initBlockCollector(eventBus *wire.EventBus) chan *block.Block {
	blockChannel := make(chan *block.Block, 1)
	collector := &blockCollector{blockChannel}
	go wire.NewEventSubscriber(eventBus, collector, string(topics.Block)).Accept()
	return blockChannel
}

func (b *blockCollector) Collect(message *bytes.Buffer) error {
	blk := block.NewBlock()
	if err := blk.Decode(message); err != nil {
		return err
	}

	b.blockChannel <- blk
	return nil
}
