package processing

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// DataBroker is a processing unit responsible for handling GetData messages. It
// maintains a connection to the outgoing message queue of the peer it receives this
// message from.
type DataBroker struct {
	db           database.DB
	responseChan chan<- *bytes.Buffer
}

// NewDataBroker returns an initialized DataBroker.
func NewDataBroker(db database.DB, responseChan chan<- *bytes.Buffer) *DataBroker {
	return &DataBroker{
		db:           db,
		responseChan: responseChan,
	}
}

// SendItems takes a GetData message from the wire, and iterates through the list,
// sending back each item's complete data to the requesting peer.
func (d *DataBroker) SendItems(m *bytes.Buffer) error {
	msg := &peermsg.Inv{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	for _, obj := range msg.InvList {
		// support only InvTypeBlock for now
		// TODO: Add functionality for InvTypeTX
		if obj.Type != peermsg.InvTypeBlock {
			continue
		}

		// Fetch block from local state. It must be available
		b, err := d.fetchBlock(obj.Hash)
		if err != nil {
			return err
		}

		// Send the block data back to the initiator node as topics.Block msg
		buf, err := marshalBlock(b)
		if err != nil {
			return err
		}

		d.responseChan <- buf
	}

	return nil
}

// TODO: This could be part of database transaction layer API
func (d *DataBroker) fetchBlock(hash []byte) (*block.Block, error) {
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

func marshalBlock(b *block.Block) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := b.Encode(buf); err != nil {
		return nil, err
	}

	return wire.AddTopic(buf, topics.Block)
}
