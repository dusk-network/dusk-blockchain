package payload

import (
	"errors"
	"io"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

type Block struct {
	Header *BlockHeader
	Txs    []merkletree.Payload
}

func NewBlock() *Block {
	return &Block{
		Header: &BlockHeader{},
	}
}

func (b *Block) SetPrevBlock(prevBlock *Block) {
	b.Header.PrevBlock = prevBlock.Header.Hash
	// Set seed here once we know more about it
}

func (b *Block) SetTime() {
	b.Header.Timestamp = time.Now().Unix()
}

func (b *Block) SetRoot() error {
	tree, err := merkletree.NewTree(b.Txs)
	if err != nil {
		return err
	}

	b.Header.TxRoot = tree.MerkleRoot
	return nil
}

func (b *Block) AddTx(tx *transactions.Stealth) {
	b.Txs = append(b.Txs, tx)
}

func (b *Block) AddCertImage(cert *Certificate) error {
	if cert.Hash == nil {
		if err := cert.SetHash(); err != nil {
			return err
		}
	}

	b.Header.CertImage = cert.Hash
	return nil
}

func (b *Block) Clear() {
	b.Header = &BlockHeader{}
	b.Txs = nil
}

func (b *Block) SetHash() error {
	return b.Header.SetHash()
}

func (b *Block) Encode(w io.Writer) error {
	if err := b.Header.Encode(w); err != nil {
		return err
	}

	lTxs := uint64(len(b.Txs))
	if err := encoding.WriteVarInt(w, lTxs); err != nil {
		return err
	}

	for _, v := range b.Txs {
		tx, ok := v.(*transactions.Stealth)
		if !ok {
			return errors.New("non-tx object found in block txs array")
		}

		if err := tx.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

func (b *Block) Decode(r io.Reader) error {
	b.Header = &BlockHeader{}
	if err := b.Header.Decode(r); err != nil {
		return err
	}

	lTxs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	b.Txs = make([]merkletree.Payload, lTxs)
	for i := uint64(0); i < lTxs; i++ {
		tx := &transactions.Stealth{}
		if err := tx.Decode(r); err != nil {
			return err
		}

		b.Txs[i] = tx
	}

	return nil
}
