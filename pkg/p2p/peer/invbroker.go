package peer

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type invBroker struct {
	gossip *processing.Gossip
	db     database.DB
	conn   *Connection
}

// TODO: Consider moving this and blockBroker to a single struct
// TODO: Consider utilizing RPCBus e.g rpcBus.Call(wire.GetMissingObjects)
func newInvBroker(conn *Connection, db database.DB) (*invBroker, error) {
	return &invBroker{
		gossip: processing.NewGossip(conn.magic),
		db:     db,
		conn:   conn,
	}, nil
}

func (b *invBroker) handleMsg(m *bytes.Buffer) error {

	msg := &peermsg.Inv{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	dataList := make([]peermsg.InvVect, 0)
	for _, obj := range msg.InvList {

		// support only InvTypeBlock for now
		if obj.Type != peermsg.InvTypeBlock {
			continue
		}

		// Check if local blockchain state does include this block hash ...
		err := b.db.View(func(t database.Transaction) error {

			_, err := t.FetchBlockExists(obj.Hash)
			if err == database.ErrBlockNotFound {
				// .. if not, let's request the full block data from the InvMsg initiator node
				dataList = append(dataList, obj)
				return nil
			}

			return err
		})

		if err != nil {
			return err
		}
	}

	if len(dataList) > 0 {
		// we've got objects that are missing, then packet and request them
		b.packAndSend(dataList)
	}

	return nil
}

func (b *invBroker) packAndSend(list []peermsg.InvVect) error {

	// Construct and encode GetData message
	// It reuses the peermsg.Inv payload structure

	msg := &peermsg.Inv{InvList: list}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	bufWithTopic, err := wire.AddTopic(buf, topics.GetData)
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
