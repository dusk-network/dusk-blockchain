package block

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header *Header
	Txs    []transactions.Transaction
}

// NewBlock will return an empty Block with an empty BlockHeader.
func NewBlock() *Block {
	return &Block{
		Header: &Header{
			Version:     0x00,
			Certificate: EmptyCertificate(),
		},
	}
}

// SetPrevBlock will set all the previous block hash field from a header.
func (b *Block) SetPrevBlock(prevHeader *Header) {
	b.Header.PrevBlockHash = prevHeader.Hash
}

// SetRoot will set the block merkle root hash.
func (b *Block) SetRoot() error {

	// convert Transaction interface to Payload interface
	var txs []merkletree.Payload
	for _, tx := range b.Txs {
		txs = append(txs, tx)
	}

	tree, err := merkletree.NewTree(txs)
	if err != nil {
		return err
	}

	b.Header.TxRoot = tree.MerkleRoot
	return nil
}

// AddTx will add a transaction to the block.
func (b *Block) AddTx(tx transactions.Transaction) {
	b.Txs = append(b.Txs, tx)
}

// Clear will empty out all the block's fields.
func (b *Block) Clear() {
	b.Header = &Header{}
	b.Txs = nil
}

// SetHash will set the block hash.
func (b *Block) SetHash() error {
	return b.Header.SetHash()
}

// Encode a Block struct and write to w.
func (b *Block) Encode(w io.Writer) error {
	if err := b.Header.Encode(w); err != nil {
		return err
	}

	lTxs := uint64(len(b.Txs))
	if err := encoding.WriteVarInt(w, lTxs); err != nil {
		return err
	}

	for _, tx := range b.Txs {
		if err := tx.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

// Decode a Block struct from r into b.
func (b *Block) Decode(r io.Reader) error {
	b.Header = &Header{}
	if err := b.Header.Decode(r); err != nil {
		return err
	}

	lTxs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	b.Txs, err = transactions.FromReader(r, lTxs)
	if err != nil {
		return err
	}

	return nil
}

// Equals returns true if two blocks are equal
func (b *Block) Equals(other *Block) bool {
	if other == nil {
		return false
	}

	if !b.Header.Equals(other.Header) {
		return false
	}

	if len(b.Txs) != len(other.Txs) {
		return false
	}

	for i := range b.Txs {
		tx := b.Txs[i]
		otherTx := other.Txs[i]

		if !tx.Equals(otherTx) {
			return false
		}
	}

	return true
}
