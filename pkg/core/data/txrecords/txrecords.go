// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package txrecords

import (
	"bytes"
	"encoding/binary"
	"log"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

// Direction is an enum which tells us whether a transaction is
// incoming or outgoing.
type Direction uint8

const (
	// In identifies incoming transaction.
	In Direction = iota
	// Out identifies outgoing transaction.
	Out
)

// TxMeta is a convenience structure which groups fields common to the TxView
// and the TxRecord.
type TxMeta struct {
	Height    uint64
	Direction Direction
	Timestamp int64
}

// TxRecord encapsulates the data stored on the DB related to a transaction such as block
// height, direction, etc. The fields related to the ContractCall are embedded
// inside the Transaction carrying the information about the `Fee` (gas) and the
// `Amount` (which can be zero for smart contract calls not related to transfer
// o value).
type TxRecord struct {
	TxMeta
	Transaction transactions.ContractCall
}

// TxView is the UX-friendly structure to be served to the UI requesting them.
type TxView struct {
	TxMeta
	Type       transactions.TxType
	Amount     uint64
	Fee        uint64
	Timelock   uint64
	Hash       []byte
	Data       []byte
	Obfuscated bool
}

// New creates a TxRecord.
func New(call transactions.ContractCall, height uint64, direction Direction) *TxRecord {
	return &TxRecord{
		Transaction: call,
		TxMeta: TxMeta{
			Direction: direction,
			Timestamp: time.Now().Unix(),
			Height:    height,
		},
	}
}

// View returns a UI consumable representation of a TXRecord.
func (t TxRecord) View() TxView {
	h, err := t.Transaction.CalculateHash()
	if err != nil {
		log.Panic(err)
	}

	view := TxView{
		TxMeta: t.TxMeta,
		Type:   t.Transaction.Type(),
		Hash:   h,
	}

	view.Fee, _ = t.Transaction.Fee()

	// switch tx := t.Transaction.(type) {
	// case *transactions.BidTransaction:
	// 	view.Timelock = tx.ExpirationHeight
	// case *transactions.StakeTransaction:
	// 	view.Timelock = tx.ExpirationHeight
	// case *transactions.Transaction:
	// 	view.Data = tx.Data
	// }

	return view
}

// Encode the TxRecord into a buffer.
func Encode(b *bytes.Buffer, t *TxRecord) error {
	if err := binary.Write(b, binary.LittleEndian, t.Direction); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.Timestamp); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.Height); err != nil {
		return err
	}

	if err := transactions.Marshal(b, t.Transaction); err != nil {
		return err
	}

	return nil
}

// Decode a TxRecord from a buffer.
func Decode(b *bytes.Buffer, t *TxRecord) error {
	if err := binary.Read(b, binary.LittleEndian, &t.Direction); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.Timestamp); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.Height); err != nil {
		return err
	}

	call := new(transactions.Transaction)
	if err := transactions.Unmarshal(b, call); err != nil {
		return err
	}

	t.Transaction = *call
	return nil
}
