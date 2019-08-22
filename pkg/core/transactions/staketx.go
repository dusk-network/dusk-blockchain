package transactions

import (
	"bytes"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
)

// Stake transactions are time lock transactions that are used by nodes
// to allow them to become provisioners, and participate in the consensus protocol.
// What does the TimeLock represent?
// In a `Stake TX` it means that the Outputs, or the node's stake is locked until after the
// specified time. After this time, they will also no longer be a provisioner,
// and must re-stake to participate again.
type Stake struct {
	TimeLock
	PubKeyEd  []byte // 32 bytes
	PubKeyBLS []byte // 33 bytes
}

// NewStake will return a Stake transaction
// Given the tx version, the locktime,the fee and M
func NewStake(ver uint8, lock, fee uint64, R, pubKeyEd, pubKeyBLS []byte) (*Stake, error) {
	if len(pubKeyEd) != 32 {
		return nil, errors.New("edwards public key is not 32 bytes")
	}

	s := &Stake{
		TimeLock:  *NewTimeLock(ver, lock, fee, R),
		PubKeyEd:  pubKeyEd,
		PubKeyBLS: pubKeyBLS,
	}
	s.TxType = StakeType

	return s, nil
}

// StandardTX returns the embedded standard tx
// Implements Transaction interface.
func (s Stake) StandardTX() Standard {
	return s.Standard
}

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
// Implements merkletree.Payload interface
func (s *Stake) CalculateHash() ([]byte, error) {
	if len(s.TxID) != 0 {
		return s.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := MarshalStake(buf, s); err != nil {
		return nil, err
	}

	txid, err := hashBytes(buf)
	if err != nil {
		return nil, err
	}
	s.TxID = txid

	return s.TxID, nil
}

// Equals returns true if two Stake tx's are the same
func (s *Stake) Equals(t Transaction) bool {

	other, ok := t.(*Stake)
	if !ok {
		return false
	}

	if !s.Standard.Equals(&other.Standard) {
		return false
	}

	if !bytes.Equal(s.PubKeyEd, other.PubKeyEd) {
		return false
	}
	if !bytes.Equal(s.PubKeyBLS, other.PubKeyBLS) {
		return false
	}

	return true
}

func (s *Stake) GetOutputAmount() uint64 {
	var sAmount ristretto.Scalar
	sAmount.UnmarshalBinary(s.Outputs[0].EncryptedAmount)
	amount := sAmount.BigInt().Uint64()
	return amount
}
