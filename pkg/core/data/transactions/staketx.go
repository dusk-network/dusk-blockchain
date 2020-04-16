package transactions

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-crypto/hash"
)

// Stake encapsulates a Stake transaction
type Stake struct {
	*Timelock
	PubKeyBLS []byte
}

// NewStake creates a new Stake transaction
func NewStake(ver uint8, netPrefix byte, fee int64, lock uint64, pubKeyBLS []byte) (*Stake, error) {
	tx, err := NewTimelock(ver, netPrefix, fee, lock)
	if err != nil {
		return nil, err
	}

	tx.TxType = StakeType
	return &Stake{
		tx,
		pubKeyBLS,
	}, nil
}

// CalculateHash calculates the hash of this transaction
func (s *Stake) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalStake(buf, s); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return txid, nil
}

// StandardTx returns the Standard transaction
func (s *Stake) StandardTx() *Standard {
	return s.Standard
}

// Type returns the transaction type
func (s *Stake) Type() TxType {
	return s.TxType
}

// Prove the transaction
func (s *Stake) Prove() error {
	return s.prove(s.CalculateHash, false)
}

// Equals test the transactions equality
func (s *Stake) Equals(t Transaction) bool {
	other, ok := t.(*Stake)
	if !ok {
		return false
	}

	if !s.Timelock.Equals(other.Timelock) {
		return false
	}

	if !bytes.Equal(s.PubKeyBLS, other.PubKeyBLS) {
		return false
	}

	return true
}

// LockTime returns the lock time of this transaction
func (s *Stake) LockTime() uint64 {
	return s.Lock
}

func marshalStake(b *bytes.Buffer, s *Stake) error {
	if err := marshalTimelock(b, s.Timelock); err != nil {
		return err
	}

	if err := writeVarInt(b, uint64(len(s.PubKeyBLS))); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}
