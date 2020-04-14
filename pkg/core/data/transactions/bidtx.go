package transactions

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-crypto/hash"
)

// Bid transaction
type Bid struct {
	*Timelock
	M []byte
}

// NewBid creates a new blind bid transaction
func NewBid(ver uint8, netPrefix byte, fee int64, lock uint64, M []byte) (*Bid, error) {
	tx, err := NewTimelock(ver, netPrefix, fee, lock)
	if err != nil {
		return nil, err
	}

	tx.TxType = BidType
	return &Bid{
		tx,
		M,
	}, nil
}

// CalculateHash calculates the hash
func (b *Bid) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalBid(buf, b); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return txid, nil
}

// StandardTx returns the standard transaction form of this bid
func (b *Bid) StandardTx() *Standard {
	return b.Standard
}

// Type returns the transaction type
func (b *Bid) Type() TxType {
	return b.TxType
}

// Prove the bid to me!
func (b *Bid) Prove() error {
	return b.prove(b.CalculateHash, false)
}

// Equals tests for equality to this bid
func (b *Bid) Equals(t Transaction) bool {
	other, ok := t.(*Bid)
	if !ok {
		return false
	}

	if !b.Timelock.Equals(other.Timelock) {
		return false
	}

	if !bytes.Equal(b.M, other.M) {
		return false
	}

	return true
}

// LockTime returns the lock time for this bid
func (b *Bid) LockTime() uint64 {
	return b.Lock
}

func marshalBid(b *bytes.Buffer, bid *Bid) error {
	if err := marshalTimelock(b, bid.Timelock); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, bid.M); err != nil {
		return err
	}

	return nil
}
