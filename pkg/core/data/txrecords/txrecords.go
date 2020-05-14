package txrecords

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	log "github.com/sirupsen/logrus"
)

// Direction is an enum which tells us whether a transaction is
// incoming or outgoing.
type Direction uint8

const (
	// In identifies incoming transaction
	In Direction = iota
	// Out identifies outgoing transaction
	Out
)

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

// TxView is the UX-friendly structure to be served to the UI requesting them
type TxView struct {
	TxMeta
	Type       string
	Amount     uint64
	Fee        uint64
	Timelock   uint64
	Hash       []byte
	Data       []byte
	Obfuscated bool
}

// String returns a human-readable representation of this transaction
func (v TxView) String() string {
	return ""
}

// New creates a TxRecord
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

// View returns a UI consumable representation of a TXRecord
func (t TxRecord) View() TxView {
	h, err := t.Transaction.CalculateHash()
	if err != nil {
		log.Panic(err)
	}

	view := TxView{
		TxMeta:     t.TxMeta,
		Type:       string(t.Transaction.Type()),
		Hash:       h,
		Obfuscated: t.Transaction.Obfuscated(),
		Fee:        t.Transaction.Fees(),
	}

	switch tx := t.Transaction.(type) {
	case *transactions.BidTransaction:
		// view.LockTime = tx.ExpirationHeight
	case *transactions.StakeTransaction:
		view.Amount = tx.Amount()
		// view.LockTime = tx.ExpirationHeight
	case *transactions.Transaction:
		if !tx.Obfuscated() {
			// loop on the Outputs
			for _, out := range tx.Outputs {
				view.Amount += out.Note.TransparentValue
			}
		}
		view.Data = tx.Data
	case *transactions.DistributeTransaction:
		view.Amount = tx.TotalReward()
	}

	// view.Data = tx.Data

	// FIXME: 459 - calculate amount
	// FIXME: 459 - fill locktime
	// FIXME: 459 - add Data
	return view
}

// Encode the TxRecord into a buffer
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

// Decode a TxRecord from a buffer
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

	call, err := transactions.Unmarshal(b)
	if err != nil {
		return err
	}

	t.Transaction = call
	return nil
}
