// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// DataBroker is a processing unit responsible for handling GetData messages. It
// maintains a connection to the outgoing message queue of the peer it receives this
// message from.
type DataBroker struct {
	db     database.DB
	rpcBus *rpcbus.RPCBus
}

// NewDataBroker returns an initialized DataBroker.
func NewDataBroker(db database.DB, rpcBus *rpcbus.RPCBus) *DataBroker {
	return &DataBroker{
		db:     db,
		rpcBus: rpcBus,
	}
}

// MarshalObjects marshals requested objects by a message of type message.Inv.
func (d *DataBroker) MarshalObjects(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	msg := m.Payload().(message.Inv)
	bufs := make([]bytes.Buffer, 0, len(msg.InvList))

	for _, obj := range msg.InvList {
		switch obj.Type {
		case message.InvTypeBlock:
			// Fetch block from local state. It must be available
			var b *block.Block

			err := d.db.View(func(t database.Transaction) error {
				var err error
				b, err = t.FetchBlock(obj.Hash)
				return err
			})
			if err != nil {
				return nil, err
			}

			// Send the block data back to the initiator node as topics.Block msg
			buf, err := marshalBlock(b)
			if err != nil {
				return nil, err
			}

			bufs = append(bufs, *buf)
		case message.InvTypeMempoolTx:
			// Try to retrieve tx from local mempool state. It might not be
			// available
			txs, err := getMempoolTxs(d.rpcBus, obj.Hash)
			if err != nil {
				return nil, err
			}

			if len(txs) != 0 {
				// Send topics.Tx with the tx data back to the initiator
				buf, err := marshalTx(txs[0])
				if err != nil {
					return nil, err
				}

				bufs = append(bufs, *buf)
			}
		}
	}

	return bufs, nil
}

// MarshalMempoolTxs marshals all or subset of pending Mempool transactions.
func (d *DataBroker) MarshalMempoolTxs(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	maxItemsSent := config.Get().Mempool.MaxInvItems
	if maxItemsSent == 0 {
		// responding to topics.Mempool disabled
		return nil, errors.New("responding to topics.Mempool is disabled")
	}

	txs, err := getMempoolTxs(d.rpcBus, nil)
	if err != nil {
		return nil, err
	}

	bufs := make([]bytes.Buffer, 0, len(txs))
	msg := &message.Inv{}

	for _, tx := range txs {
		hash, calcErr := tx.CalculateHash()
		if calcErr != nil {
			continue
		}

		msg.AddItem(message.InvTypeMempoolTx, hash)

		maxItemsSent--
		if maxItemsSent == 0 {
			break
		}
	}

	buf, err := marshalInv(msg)
	if err == nil {
		bufs = append(bufs, buf)
	}
	// A txID will not be found in a few situations:
	//
	// - The node has restarted and lost this Tx
	// - The node has recently accepted a block that includes this Tx
	// No action to run in these cases.

	return bufs, nil
}

func marshalBlock(b *block.Block) (*bytes.Buffer, error) {
	// TODO: following is more efficient, saves an allocation and avoids the explicit Prepend
	// buf := topics.Topics[topics.Block].Buffer
	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, b); err != nil {
		return nil, err
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		return nil, err
	}

	return buf, nil
}

func marshalTx(tx transactions.ContractCall) (*bytes.Buffer, error) {
	// TODO: following is more efficient, saves an allocation and avoids the explicit Prepend
	// buf := topics.Topics[topics.Block].Buffer
	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, tx); err != nil {
		return nil, err
	}

	if err := topics.Prepend(buf, topics.Tx); err != nil {
		return nil, err
	}

	return buf, nil
}
