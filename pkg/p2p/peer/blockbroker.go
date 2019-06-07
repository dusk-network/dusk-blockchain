package peer

import (
	"bytes"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type blockBroker struct {
	gossip *processing.Gossip
	db     database.DB
	conn   *Connection
}

func newBlockBroker(conn *Connection) (*blockBroker, error) {
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		return nil, err
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), true)
	if err != nil {
		return nil, err
	}

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
	blocks, err := b.fetchBlocks(msg)
	if err != nil {
		return err
	}

	for _, blk := range blocks {
		if err := b.sendBlock(blk); err != nil {
			return err
		}
	}

	return nil
}

func (b *blockBroker) fetchBlocks(msg *peermsg.GetBlocks) ([]*block.Block, error) {
	var blocks []*block.Block
	err := b.db.View(func(t database.Transaction) error {
		header, err := t.FetchBlockHeader(msg.Locators[0])
		if err != nil {
			return err
		}

		height := header.Height
		for {
			height++
			blk, err := reconstructBlock(t, height)
			if err != nil {
				return err
			}

			if blk == nil {
				// This means we hit the end of the chain, so we can just return.
				return nil
			}

			blocks = append(blocks, blk)
		}
	})

	return blocks, err
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

func reconstructBlock(t database.Transaction, height uint64) (*block.Block, error) {
	hash, err := t.FetchBlockHashByHeight(height)
	if err != nil {
		return nil, nil
	}

	header, err := t.FetchBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	txs, err := t.FetchBlockTxs(hash)
	if err != nil {
		return nil, err
	}

	return &block.Block{
		Header: header,
		Txs:    txs,
	}, nil
}
