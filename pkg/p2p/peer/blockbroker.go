package peer

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type blockBroker struct {
	gossip *processing.Gossip
	db     database.DB
	conn   *Connection
}

func newBlockBroker(conn *Connection, db database.DB) (*blockBroker, error) {
	return &blockBroker{
		gossip: processing.NewGossip(conn.magic),
		db:     db,
		conn:   conn,
	}, nil
}

func (b *blockBroker) sendBlocks(m *bytes.Buffer) error {
	msg := &peermsg.GetBlocks{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	// Fetch blocks to send to peer, going off his Locator
	height, err := b.fetchLocatorHeight(msg)
	if err != nil {
		return err
	}

	for {
		height++

		blk, err := b.reconstructBlock(height)
		if err != nil {
			return err
		}

		// We don't send the target block, as the peer should already
		// have this in memory.
		if bytes.Equal(blk.Header.Hash, msg.Target) {
			return nil
		}

		if err := b.sendBlock(blk); err != nil {
			return err
		}
	}
}

func (b *blockBroker) fetchLocatorHeight(msg *peermsg.GetBlocks) (uint64, error) {
	var height uint64
	err := b.db.View(func(t database.Transaction) error {
		header, err := t.FetchBlockHeader(msg.Locators[0])
		if err != nil {
			return err
		}

		height = header.Height
		return nil
	})

	return height, err
}

func (b *blockBroker) sendBlock(blk *block.Block) error {
	buf := new(bytes.Buffer)
	if err := blk.Encode(buf); err != nil {
		return err
	}

	bufWithTopic, err := wire.AddTopic(buf, topics.Block)
	if err != nil {
		return err
	}

	encodedMsg, err := b.gossip.Process(bufWithTopic)
	if err != nil {
		return err
	}

	if _, err := b.conn.Write(encodedMsg.Bytes()); err != nil {
		return err
	}

	return nil
}

func (b *blockBroker) reconstructBlock(height uint64) (*block.Block, error) {
	var blk *block.Block
	err := b.db.View(func(t database.Transaction) error {
		hash, err := t.FetchBlockHashByHeight(height)
		if err != nil {
			return err
		}

		header, err := t.FetchBlockHeader(hash)
		if err != nil {
			return err
		}

		txs, err := t.FetchBlockTxs(hash)
		if err != nil {
			return err
		}

		blk = &block.Block{
			Header: header,
			Txs:    txs,
		}
		return nil
	})

	return blk, err
}
