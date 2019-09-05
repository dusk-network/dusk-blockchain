package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-crypto/hash"
)

type Stake struct {
	*Timelock
	PubKeyEd  []byte
	PubKeyBLS []byte
}

func NewStake(ver uint8, netPrefix byte, fee int64, lock uint64, pubKeyEd, pubKeyBLS []byte) (*Stake, error) {
	tx, err := NewTimelock(ver, netPrefix, fee, lock)
	if err != nil {
		return nil, err
	}

	tx.TxType = StakeType
	return &Stake{
		tx,
		pubKeyEd,
		pubKeyBLS,
	}, nil
}

func (s *Stake) CalculateHash() ([]byte, error) {
	if len(s.TxID) != 0 {
		return s.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := marshalStake(buf, s, false); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	s.TxID = txid
	return txid, nil
}

func (s *Stake) StandardTx() Standard {
	return *s.Standard
}

func (s *Stake) Type() TxType {
	return s.TxType
}

func (s *Stake) Prove() error {
	return s.prove(s.CalculateHash, false)
}

func (s *Stake) Equals(t Transaction) bool {
	other, ok := t.(*Stake)
	if !ok {
		return false
	}

	if !s.Timelock.Equals(other.Timelock) {
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
