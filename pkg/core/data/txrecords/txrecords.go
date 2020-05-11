package txrecords

import (
	"bytes"
	"encoding/binary"
	"time"

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
// height, direction, amount, etc
type TxRecord struct {
	Direction
	Timestamp int64
	Height    uint64
	transactions.TxType
	Amount   uint64 // can be zero for smart contract calls not related to transfer of value
	Fee      uint64 // is essentially gas
	Data     []byte // binary  representation of the smart contract call inputs
	LockTime uint64 // expressed in blockheight terms
	transactions.ContractCall
}

// New creates a TxRecord
func New(tx transactions.ContractCall, height uint64, direction Direction) *TxRecord {
	// FIXME: 459 translate the ContractCall into the above struct
	return &TxRecord{
		Direction:    direction,
		Timestamp:    time.Now().Unix(),
		Height:       height,
		TxType:       tx.Type(),
		ContractCall: tx,
	}
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

	if err := transactions.Marshal(b, t.ContractCall); err != nil {
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

	if err := binary.Read(b, binary.LittleEndian, &t.TxType); err != nil {
		return err
	}

	var call transactions.ContractCall
	switch t.TxType {
	case transactions.Tx:
		call = new(transactions.Transaction)
	case transactions.Distribute:
		call = new(transactions.DistributeTransaction)
	case transactions.WithdrawFees:
		call = new(transactions.WithdrawFeesTransaction)
	case transactions.Bid:
		call = new(transactions.BidTransaction)
	case transactions.Stake:
		call = new(transactions.StakeTransaction)
	case transactions.Slash:
		call = new(transactions.SlashTransaction)
	case transactions.WithdrawStake:
		call = new(transactions.WithdrawStakeTransaction)
	case transactions.WithdrawBid:
		call = new(transactions.WithdrawBidTransaction)
	}

	if err := transactions.Unmarshal(b, call); err != nil {
		return err
	}

	t.ContractCall = call
	return nil
}
