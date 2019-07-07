package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// TimeLock represents a standard transaction that has an additional time restriction
// What does the time-lock represent?
// For a `Standard TimeLock`; that the TX can only become valid after the time stated.
// This is not the case for others, please check each transaction for the significance of the timelock
type TimeLock struct {
	Standard
	// Lock represents the value for the timelock.
	// There are 2^64 possible values.
	// The Lock can be specified in either blockheight or unix timestamp
	// 0 to 0x8000000000000000 will represent the unixtimestamp in seconds
	// >= 0x8000000000000000 will represnt a blockheight
	// This means that block zero can be represented as 0x8000000000000000
	// And max blockheight for time-lock is `2^64-0x8000000000000000`
	Lock uint64
}

// TimeLockBlockZero is the integer that represents the blockHeight of zero for
// a timelock transaction
const TimeLockBlockZero = 0x8000000000000000
const MaxLockTime = 250000

// NewTimeLock will return a TimeLock transaction
// Given the tx version, the locktime and the fee
func NewTimeLock(ver uint8, lock, fee uint64, R []byte) *TimeLock {
	t := &TimeLock{
		Standard: *NewStandard(ver, fee, R),
		Lock:     lock,
	}
	t.TxType = TimelockType
	return t
}

// Encode implements the Encoder interface
func (t *TimeLock) Encode(w io.Writer) error {
	if err := t.Standard.Encode(w); err != nil {
		return err
	}
	if err := encoding.WriteUint64(w, binary.LittleEndian, t.Lock); err != nil {
		return err
	}
	return nil
}

// Decode implements the Decoder interface
func (t *TimeLock) Decode(r io.Reader) error {

	if err := t.Standard.Decode(r); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &t.Lock); err != nil {
		return err
	}
	return nil
}

// StandardTX returns the embedded standard tx
// Implements Transaction interface.
func (t TimeLock) StandardTX() Standard {
	return t.Standard
}

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
//// Implements merkletree.Payload interface
func (t *TimeLock) CalculateHash() ([]byte, error) {
	if len(t.TxID) != 0 {
		return t.TxID, nil
	}

	txid, err := hashBytes(t.Encode)
	if err != nil {
		return nil, err
	}
	t.TxID = txid

	return t.TxID, nil
}

// Equals returns true if two timelocks tx's are the same
func (t *TimeLock) Equals(tr Transaction) bool {

	other, ok := tr.(*TimeLock)
	if !ok {
		return false
	}

	if !t.Standard.Equals(&other.Standard) {
		return false
	}

	if t.Lock != other.Lock {
		return false
	}

	return true
}
