package transactions

import (
	"encoding/binary"
	"io"

	wiretx "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

type TimelockTx struct {
	*StandardTx
	Lock uint64
}

func NewTimeLockTx(netPrefix byte, fee int64, lock uint64) (*TimelockTx, error) {
	tx, err := NewStandard(netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return &TimelockTx{
		tx,
		lock,
	}, nil
}

func (tl *TimelockTx) Hash() ([]byte, error) {
	return hashBytes(tl.encode)
}

func (tl *TimelockTx) encode(w io.Writer, encodeSig bool) error {
	if err := tl.StandardTx.encode(w, encodeSig); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, tl.Lock)
}

func (tl *TimelockTx) Prove() error {
	return tl.prove(tl.Hash, true)
}

func (tl *TimelockTx) Encode(w io.Writer) error {
	return tl.encode(w, true)
}

func (tl *TimelockTx) Decode(r io.Reader) error {
	tl.StandardTx = &StandardTx{}
	if err := tl.StandardTx.Decode(r); err != nil {
		return err
	}
	return binary.Read(r, binary.BigEndian, &tl.Lock)
}

func (tl *TimelockTx) WireTimeLockTx() (*wiretx.TimeLock, error) {

	standardTx, err := tl.StandardTx.WireStandardTx()
	if err != nil {
		return nil, err
	}
	tx := &wiretx.TimeLock{
		*standardTx,
		tl.Lock,
	}
	tx.TxType = wiretx.TimelockType
	return tx, nil
}
