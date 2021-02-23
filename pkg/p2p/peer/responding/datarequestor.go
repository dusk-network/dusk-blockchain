// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// DataRequestor is a processing unit which handles inventory messages received from peers
// on the Dusk wire protocol. It maintains a connection to the outgoing message queue
// of an individual peer.
type DataRequestor struct {
	db     database.DB
	rpcBus *rpcbus.RPCBus
	// The DataRequestor maintains a separate instance of the dupemap,
	// to ensure advertised hashes are not followed up on more than once.
	dupemap *dupemap.DupeMap

	// This mutex is used to ensure that blocks are requested in a serial
	// manner. If multiple goroutines are constructing `GetData` messages,
	// there might be a race condition between the two in which they construct
	// `GetData` messages that are spotty. This will cause annoying behavior
	// during sync, such as flooding the network with more requests than is
	// necessary.
	lock sync.Mutex
}

// NewDataRequestor returns an initialized DataRequestor.
func NewDataRequestor(db database.DB, rpcBus *rpcbus.RPCBus, broker eventbus.Broker) *DataRequestor {
	return &DataRequestor{
		db:      db,
		rpcBus:  rpcBus,
		dupemap: dupemap.Launch(broker),
	}
}

// RequestMissingItems takes an inventory message, checks it for any items that the node
// is missing, puts these items in a GetData wire message, and sends it off to the peer's
// outgoing message queue, requesting the items in full.
func (d *DataRequestor) RequestMissingItems(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	msg := m.Payload().(message.Inv)
	getData := &message.Inv{}

	for _, obj := range msg.InvList {
		if !d.dupemap.CanFwd(bytes.NewBuffer(obj.Hash)) {
			continue
		}

		switch obj.Type {
		case message.InvTypeBlock:
			// Check if local blockchain state does include this block hash ...
			err := d.db.View(func(t database.Transaction) error {
				_, err := t.FetchBlockExists(obj.Hash)
				if err == database.ErrBlockNotFound {
					// .. if not, let's request the full block data from the InvMsg initiator node
					getData.AddItem(message.InvTypeBlock, obj.Hash)
					return nil
				}

				return err
			})
			if err != nil {
				return nil, err
			}
		case message.InvTypeMempoolTx:
			txs, _ := getMempoolTxs(d.rpcBus, obj.Hash)

			// TxID not found in the local mempool:
			// it migth be due to a few reasons:
			//
			// Tx has never included in this mempool,
			// Tx has been included in this mempool but lost on a suddent restart
			// Tx has been already accepted.
			// TODO: To check that look for this Tx in the last 10 blocks (db.FetchTxExists())
			if len(txs) == 0 {
				getData.AddItem(message.InvTypeMempoolTx, obj.Hash)
			}
		}
	}

	// If we found any items to be missing, we request them from the peer who
	// advertised them.
	if getData.InvList != nil {
		// we've got objects that are missing, then packet and request them
		buf, err := marshalGetData(getData)
		return []bytes.Buffer{*buf}, err
	}

	return nil, nil
}

func marshalGetData(getData *message.Inv) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := getData.Encode(buf); err != nil {
		log.Panic(err)
	}

	if err := topics.Prepend(buf, topics.GetData); err != nil {
		return nil, err
	}

	return buf, nil
}

// getMempoolTxs is a wire.GetMempoolTx API wrapper. Later it could be moved into
// a separate utils pkg.
func getMempoolTxs(bus *rpcbus.RPCBus, txID []byte) ([]transactions.ContractCall, error) {
	buf := new(bytes.Buffer)
	_, _ = buf.Write(txID)

	timeoutGetMempoolTXs := time.Duration(config.Get().Timeout.TimeoutGetMempoolTXs) * time.Second

	resp, err := bus.Call(topics.GetMempoolTxs, rpcbus.NewRequest(*buf), timeoutGetMempoolTXs)
	if err != nil {
		return nil, err
	}

	mempoolTxs := resp.([]transactions.ContractCall)
	return mempoolTxs, nil
}
