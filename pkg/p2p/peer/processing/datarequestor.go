package processing

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// DataRequestor is a processing unit which handles inventory messages received from peers
// on the Dusk wire protocol. It maintains a connection to the outgoing message queue
// of an individual peer.
type DataRequestor struct {
	db           database.DB
	responseChan chan<- *bytes.Buffer
	rpcBus       *wire.RPCBus
}

// NewDataRequestor returns an initialized DataRequestor.
func NewDataRequestor(db database.DB, rpcBus *wire.RPCBus, responseChan chan<- *bytes.Buffer) *DataRequestor {
	return &DataRequestor{
		db:           db,
		responseChan: responseChan,
		rpcBus:       rpcBus,
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

		switch obj.Type {
		case peermsg.InvTypeBlock:

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

		case peermsg.InvTypeMempoolTx:

			txs, _ := GetMempoolTxs(d.rpcBus, obj.Hash)
			if len(txs) == 0 {
				// TxID not found in the local mempool:

				// it migth be due to a few reasons:
				//
				// Tx has never included in this mempool,
				// Tx has been included in this mempool but lost on a suddent restart
				// Tx has been already accepted.
				// TODO: To check that look for this Tx in the last 10 blocks (db.FetchTxExists())
				getData.AddItem(peermsg.InvTypeMempoolTx, obj.Hash)
			}
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

// RequestMempoolItems sends topics.Mempool to request available mempool txs
func (d *DataRequestor) RequestMempoolItems() error {

	buf, err := wire.AddTopic(new(bytes.Buffer), topics.MemPool)
	if err != nil {
		return err
	}
	d.responseChan <- buf
	return nil
}

func marshalGetData(getData *peermsg.Inv) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := getData.Encode(buf); err != nil {
		panic(err)
	}

	return wire.AddTopic(buf, topics.GetData)
}

// GetMempoolTxs is a wire.GetMempoolTx API wrapper. Later it could be moved into
// a separate utils pkg
func GetMempoolTxs(bus *wire.RPCBus, txID []byte) ([]transactions.Transaction, error) {

	buf := new(bytes.Buffer)
	buf.Write(txID)
	r, err := bus.Call(wire.GetMempoolTxs, wire.NewRequest(*buf, 3))
	if err != nil {
		return nil, err
	}

	lTxs, err := encoding.ReadVarInt(&r)
	if err != nil {
		return nil, err
	}

	mempoolTxs, err := transactions.FromReader(&r, lTxs)
	if err != nil {
		return nil, err
	}

	return mempoolTxs, nil
}
