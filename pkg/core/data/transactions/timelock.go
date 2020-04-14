package transactions

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-crypto/hash"
)

// MaxLockTime sets the maximum lock time to 250000 blocks
const MaxLockTime = 250000

// GenesisExpirationHeight sets the heigth for expiration of the Genesis block
const GenesisExpirationHeight = 250001

// Timelock represents a standard transaction that has an additional time restriction
// What does the time-lock represent?
// For a `Standard TimeLock`; that the TX can only become valid after the time stated.
// This is not the case for others, please check each transaction for the significance of the timelock
type Timelock struct {
	*Standard
	Lock uint64
}

// NewTimelock creates a Timelock
func NewTimelock(ver uint8, netPrefix byte, fee int64, lock uint64) (*Timelock, error) {
	tx, err := NewStandard(ver, netPrefix, fee)
	if err != nil {
		return nil, err
	}

	tx.TxType = TimelockType
	return &Timelock{
		tx,
		lock,
	}, nil
}

// CalculateHash calculate the SHA3 hash of the timelock
func (tl *Timelock) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalTimelock(buf, tl); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return txid, nil
}

// StandardTx returns the standard transaction encapsulated within the TimeLock
func (tl *Timelock) StandardTx() *Standard {
	return tl.Standard
}

// Type returns the transaction type
func (tl *Timelock) Type() TxType {
	return tl.TxType
}

// Prove by checking that the hash is correct
func (tl *Timelock) Prove() error {
	return tl.prove(tl.CalculateHash, true)
}

// Equals checks the equality of the transaction
func (tl *Timelock) Equals(t Transaction) bool {
	other, ok := t.(*Timelock)
	if !ok {
		return false
	}

	if !tl.Standard.Equals(other.Standard) {
		return false
	}

	return true
}

// LockTime returns the remaining lock time
func (tl *Timelock) LockTime() uint64 {
	return tl.Lock
}

func marshalTimelock(b *bytes.Buffer, tl *Timelock) error {
	if err := marshalStandard(b, tl.Standard); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, tl.Lock); err != nil {
		return err
	}

	return nil
}
