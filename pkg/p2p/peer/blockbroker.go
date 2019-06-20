package peer

import (
	"bytes"

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

func newBlockBroker(conn *Connection, db database.DB) *blockBroker {
	return &blockBroker{
		gossip: processing.NewGossip(conn.magic),
		db:     db,
		conn:   conn,
	}
}

// Send back the set of block hashes between the message locator and target.
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

	// Fill an inv message with all block hashes between the locator
	// and the chain tip.
	inv := &peermsg.Inv{}
	for {
		height++

		var hash []byte
		err := b.db.View(func(t database.Transaction) error {
			hash, err = t.FetchBlockHashByHeight(height)
			return err
		})

		if err != nil {
			return err
		}

		// We don't send the target block, as the peer should already
		// have this in memory.
		if bytes.Equal(hash, msg.Target) {
			break
		}

		inv.AddItem(peermsg.InvTypeBlock, hash)
	}

	return b.sendInv(inv)
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

func (b *blockBroker) sendInv(msg *peermsg.Inv) error {
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		return err
	}

	bufWithTopic, err := wire.AddTopic(buf, topics.Inv)
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
