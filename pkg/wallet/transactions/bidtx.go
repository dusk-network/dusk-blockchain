package transactions

import (
	"encoding/binary"
	"io"

	wiretx "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

type BidTx struct {
	*TimelockTx
	M []byte
}

func NewBidTx(netPrefix byte, fee int64, lock uint64, M []byte) (*BidTx, error) {
	tx, err := NewTimeLockTx(netPrefix, fee, lock)
	if err != nil {
		return nil, err
	}

	return &BidTx{
		tx,
		M,
	}, nil
}

func (b *BidTx) Hash() ([]byte, error) {
	return hashBytes(b.encode)
}

func (b *BidTx) encode(w io.Writer, encodeSig bool) error {
	if err := b.TimelockTx.encode(w, encodeSig); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, b.M)
}

func (b *BidTx) Prove() error {
	return b.prove(b.Hash, false)
}

func (b *BidTx) Encode(w io.Writer) error {
	return b.encode(w, true)
}

func (b *BidTx) Decode(r io.Reader) error {
	b.TimelockTx = &TimelockTx{}

	if err := b.TimelockTx.Decode(r); err != nil {
		return err
	}

	var MBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &MBytes)
	if err != nil {
		return err
	}
	copy(b.M, MBytes[:])

	return nil
}

func (b *BidTx) WireBid() (*wiretx.Bid, error) {

	tl, err := b.WireTimeLockTx()
	if err != nil {
		return nil, err
	}

	tx := &wiretx.Bid{
		*tl,
		b.M,
	}
	tx.TxType = wiretx.BidType

	return tx, nil
}
