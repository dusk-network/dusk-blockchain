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
	var header *block.Header
	err := b.db.View(func(t database.Transaction) error {
		// TODO: this needs to be replaced with a `FetchBlocks` function or something
		// similar.
		var err error
		header, err = t.FetchBlockHeader([]byte{})
		return err
	})

	if err != nil {
		return err
	}

	// TODO: should be block instead of header
	buf := new(bytes.Buffer)
	if err := header.Encode(buf); err != nil {
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
