package chain

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	blockCollector struct {
		blockChannel chan<- *block.Block
	}

	txCollector struct {
		txChannel chan<- transactions.Transaction
	}
)

func initBlockCollector(eventBus *wire.EventBus) chan *block.Block {
	blockChannel := make(chan *block.Block, 1)
	collector := &blockCollector{blockChannel}
	go wire.NewTopicListener(eventBus, collector, string(topics.Block)).Accept()
	return blockChannel
}

func initTxCollector(eventBus *wire.EventBus) chan transactions.Transaction {
	txChannel := make(chan transactions.Transaction, 10)
	collector := &txCollector{txChannel}
	go wire.NewTopicListener(eventBus, collector, string(topics.Tx)).Accept()
	return txChannel
}

func (b *blockCollector) Collect(message *bytes.Buffer) error {
	blk := block.NewBlock()
	if err := blk.Decode(message); err != nil {
		return err
	}

	b.blockChannel <- blk
	return nil
}

func (t *txCollector) Collect(message *bytes.Buffer) error {
	txs, err := transactions.FromReader(message, 1)
	if err != nil {
		return err
	}

	t.txChannel <- txs[0]
	return nil
}
