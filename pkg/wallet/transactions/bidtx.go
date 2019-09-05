package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-crypto/hash"
)

type Bid struct {
	*Timelock
	M []byte
}

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

func (b *Bid) CalculateHash() ([]byte, error) {
	if len(b.TxID) != 0 {
		return b.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := marshalBid(buf, b, false); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	b.TxID = txid
	return txid, nil
}

func (b *Bid) StandardTx() Standard {
	return *b.Standard
}

func (b *Bid) Type() TxType {
	return b.TxType
}

func (b *Bid) Prove() error {
	return b.prove(b.CalculateHash, false)
}

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
