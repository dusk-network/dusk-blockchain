package txrecords

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io/ioutil"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
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

// TxRecord encapsulates the data stored on the DB related to a transaction such as block
// heigth, direction, amount, etc
type TxRecord struct {
	Direction
	Timestamp int64
	Height    uint64
	transactions.TxType
	Amount       uint64
	UnlockHeight uint64
	Recipient    string
}

// New creates a TxRecord
func New(tx transactions.Transaction, height uint64, direction Direction, privView *key.PrivateView) *TxRecord {
	t := &TxRecord{
		Direction:    direction,
		Timestamp:    time.Now().Unix(),
		Height:       height,
		TxType:       tx.Type(),
		Amount:       tx.StandardTx().Outputs[0].EncryptedAmount.BigInt().Uint64(),
		UnlockHeight: height + tx.LockTime(),
		Recipient:    hex.EncodeToString(tx.StandardTx().Outputs[0].PubKey.P.Bytes()),
	}

	if transactions.ShouldEncryptValues(tx) {
		amountScalar := transactions.DecryptAmount(tx.StandardTx().Outputs[0].EncryptedAmount, tx.StandardTx().R, 0, *privView)
		t.Amount = amountScalar.BigInt().Uint64()
	}
	return t
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

	if err := binary.Write(b, binary.LittleEndian, t.TxType); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.Amount); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.UnlockHeight); err != nil {
		return err
	}

	_, err := b.Write([]byte(t.Recipient))
	return err
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

	if err := binary.Read(b, binary.LittleEndian, &t.TxType); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.Amount); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.UnlockHeight); err != nil {
		return err
	}

	recipientBytes, err := ioutil.ReadAll(b)
	if err != nil {
		return err
	}

	t.Recipient = string(recipientBytes)
	return nil
}
