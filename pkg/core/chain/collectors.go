package chain

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	blockCollector struct {
		blockChannel chan<- *block.Block
	}

	hashCollector struct {
		hashChannel chan []byte
	}
)

// Init a block collector compatible with topics.Block and topics.Candidate
func initBlockCollector(eventBus *wire.EventBus, topic string) chan *block.Block {
	blockChannel := make(chan *block.Block, 1)
	collector := &blockCollector{blockChannel}
	go wire.NewTopicListener(eventBus, collector, topic).Accept()
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

func initWinningHashCollector(eventBus *wire.EventBus) chan []byte {
	hashChannel := make(chan []byte, 1)
	collector := &hashCollector{hashChannel}
	go wire.NewTopicListener(eventBus, collector, msg.WinningBlockTopic).Accept()
	return hashChannel
}

func (c *hashCollector) Collect(message *bytes.Buffer) error {
	c.hashChannel <- message.Bytes()
	return nil
}
