package block

import (
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/merkletree"
)

func Marshal(r io.Writer, b *Block) error {
	if err := MarshalHeader(r, b.Header); err != nil {
		return err
	}

	lenTxs := uint64(len(b.Txs))
	if err := encoding.WriteVarInt(r, lenTxs); err != nil {
		return err
	}

	// TODO: parallelize transaction serialization
	for _, tx := range b.Txs {
		if err := transactions.Marshal(r, tx); err != nil {
			return err
		}
	}

	return nil
}

func Unmarshal(r io.Reader, b *Block) error {
	if err := UnmarshalHeader(r, b.Header); err != nil {
		return err
	}

	lTxs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	b.Txs = make([]transactions.Transaction, lTxs)
	for i := range b.Txs {
		tx, err := transactions.Unmarshal(r)
		if err != nil {
			return err
		}
		b.Txs[i] = tx
	}

	return nil
}

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header *Header
	Txs    []transactions.Transaction
}

// NewBlock will return an empty Block with an empty BlockHeader.
func NewBlock() *Block {
	return &Block{
		Header: NewHeader(),
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
