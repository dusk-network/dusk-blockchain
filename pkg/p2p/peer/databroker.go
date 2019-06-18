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

type dataBroker struct {
	gossip *processing.Gossip
	db     database.DB
	conn   *Connection
}

// TODO: Consider moving this and blockBroker to a single struct
// TODO: Consider utilizing RPCBus e.g rpcBus.Call(wire.GetMissingObjects)
func newDataBroker(conn *Connection, db database.DB) (*dataBroker, error) {
	return &dataBroker{
		gossip: processing.NewGossip(conn.magic),
		db:     db,
		conn:   conn,
	}, nil
}

func (d *dataBroker) handleMsg(m *bytes.Buffer) error {

	msg := &peermsg.Inv{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	for _, obj := range msg.InvList {

		// support only InvTypeBlock for now
		if obj.Type != peermsg.InvTypeBlock {
			continue
		}

		// Fetch block from local state. It must be available
		b, err := d.fetchBlock(obj.Hash)
		if err != nil {
			return err
		}

		// Send the block data back to the initiator node as topics.Block msg
		if err := d.packAndSend(b); err != nil {
			return nil
		}
	}

	return nil
}

// TODO: This could be part of database transaction layer API
func (d *dataBroker) fetchBlock(hash []byte) (*block.Block, error) {

	var blk *block.Block
	err := d.db.View(func(t database.Transaction) error {
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

func (d *dataBroker) packAndSend(b *block.Block) error {
	buf := new(bytes.Buffer)
	if err := b.Encode(buf); err != nil {
		return err
	}

	bufWithTopic, err := wire.AddTopic(buf, topics.Block)
	if err != nil {
		return err
	}

	encodedMsg, err := d.gossip.Process(bufWithTopic)
	if err != nil {
		return err
	}

	if _, err := d.conn.Write(encodedMsg.Bytes()); err != nil {
		return err
	}

	return nil
}
