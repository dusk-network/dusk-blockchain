package transactions

import (
	"bytes"
	"errors"
)

// Bid represents the bidding transaction
// that is used within the blindbid procedure
// What does the TimeLock represent?
// In a `Bidding TX` it means that the outputs are locked until after the specified time
type Bid struct {
	TimeLock
	// M represents the pre-image of the secret value k.
	// M = H(K)
	M []byte // 32 Byte
}

// NewBid will return a Bid transaction
// Given the tx version, the locktime,the fee and M
func NewBid(ver uint8, lock, fee uint64, R, M []byte) (*Bid, error) {
	if len(M) != 32 {
		return nil, errors.New("m is not 32 bytes")
	}

	b := &Bid{
		TimeLock: *NewTimeLock(ver, lock, fee, R),
		M:        M,
	}
	b.TxType = BidType

	return b, nil
}

// StandardTX returns the embedded standard tx
// Implements Transaction interface.
func (b Bid) StandardTX() Standard {
	return b.Standard
}

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
// Implements merkletree.Payload interface
func (b *Bid) CalculateHash() ([]byte, error) {
	if len(b.TxID) != 0 {
		return b.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := MarshalBid(buf, b); err != nil {
		return nil, err
	}

	txid, err := hashBytes(buf)
	if err != nil {
		return nil, err
	}
	b.TxID = txid

	return b.TxID, nil
}

// Equals returns true if two Bid tx's are the same
func (b *Bid) Equals(t Transaction) bool {

	other, ok := t.(*Bid)
	if !ok {
		return false
	}

	if !b.Standard.Equals(&other.Standard) {
		return false
	}

	if b.Lock != other.Lock {
		return false
	}

	return true
}
