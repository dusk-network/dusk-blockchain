package transactions

import (
	"encoding/binary"
	"io"

	wiretx "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

type StakeTx struct {
	*TimelockTx
	PubKeyEd  []byte
	PubKeyBLS []byte
}

func NewStakeTx(netPrefix byte, fee int64, lock uint64, pubKeyEd, pubKeyBLS []byte) (*StakeTx, error) {

	tx, err := NewTimeLockTx(netPrefix, fee, lock)
	if err != nil {
		return nil, err
	}

	return &StakeTx{
		tx,
		pubKeyEd,
		pubKeyBLS,
	}, nil
}

func (s *StakeTx) Hash() ([]byte, error) {
	return hashBytes(s.encode)
}

func (s *StakeTx) encode(w io.Writer, encodeSig bool) error {
	if err := s.TimelockTx.encode(w, encodeSig); err != nil {
		return err
	}

	lenEd := uint64(len(s.PubKeyEd))
	if err := binary.Write(w, binary.BigEndian, lenEd); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, s.PubKeyEd); err != nil {
		return err
	}

	lenBLS := uint64(len(s.PubKeyBLS))
	if err := binary.Write(w, binary.BigEndian, lenBLS); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, s.PubKeyBLS)
}

func (s *StakeTx) Prove() error {
	return s.prove(s.Hash, false)
}

func (s *StakeTx) Encode(w io.Writer) error {
	return s.encode(w, true)
}

func (s *StakeTx) Decode(r io.Reader) error {

	s.TimelockTx = &TimelockTx{}

	if err := s.TimelockTx.Decode(r); err != nil {
		return err
	}

	var lenEd, lenBLS uint64
	if err := binary.Read(r, binary.BigEndian, &lenEd); err != nil {
		return err
	}
	s.PubKeyEd = make([]byte, lenEd)
	if err := binary.Read(r, binary.BigEndian, &s.PubKeyEd); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &lenBLS); err != nil {
		return err
	}
	s.PubKeyBLS = make([]byte, lenBLS)

	return binary.Read(r, binary.BigEndian, &s.PubKeyBLS)
}

func (s *StakeTx) WireStakeTx() (*wiretx.Stake, error) {

	tl, err := s.WireTimeLockTx()
	if err != nil {
		return nil, err
	}

	tx := &wiretx.Stake{
		*tl,
		s.PubKeyEd,
		s.PubKeyBLS,
	}
	tx.TxType = wiretx.StakeType

	return tx, nil
}
