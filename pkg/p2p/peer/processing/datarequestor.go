package processing

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// DataRequestor is a processing unit which handles inventory messages received from peers
// on the Dusk wire protocol. It maintains a connection to the outgoing message queue
// of an individual peer.
type DataRequestor struct {
	db           database.DB
	responseChan chan<- *bytes.Buffer
}

// NewDataRequestor returns an initialized DataRequestor.
func NewDataRequestor(db database.DB, responseChan chan<- *bytes.Buffer) *DataRequestor {
	return &DataRequestor{
		db:           db,
		responseChan: responseChan,
	}
}

// AskForMissingItems takes an inventory message, checks it for any items that the node
// is missing, puts these items in a GetData wire message, and sends it off to the peer's
// outgoing message queue, requesting the items in full.
func (d *DataRequestor) RequestMissingItems(m *bytes.Buffer) error {
	msg := &peermsg.Inv{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	getData := &peermsg.Inv{}
	for _, obj := range msg.InvList {
		// support only InvTypeBlock for now
		if obj.Type != peermsg.InvTypeBlock {
			continue
		}

		// Check if local blockchain state does include this block hash ...
		err := d.db.View(func(t database.Transaction) error {
			_, err := t.FetchBlockExists(obj.Hash)
			if err == database.ErrBlockNotFound {
				// .. if not, let's request the full block data from the InvMsg initiator node
				getData.AddItem(peermsg.InvTypeBlock, obj.Hash)
				return nil
			}

			return err
		})

		if err != nil {
			return err
		}
	}

	// If we found any items to be missing, we request them from the peer who
	// advertised them.
	if getData.InvList != nil {
		// we've got objects that are missing, then packet and request them
		buf, err := marshalGetData(getData)
		if err != nil {
			return err
		}

		d.responseChan <- buf
	}

	return nil
}

func marshalGetData(getData *peermsg.Inv) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := getData.Encode(buf); err != nil {
		panic(err)
	}

	return wire.AddTopic(buf, topics.GetData)
}
