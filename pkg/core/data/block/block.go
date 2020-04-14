package block

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-crypto/merkletree"
)

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

// CalculateRoot will calculate and return the block merkle root hash.
func (b *Block) CalculateRoot() ([]byte, error) {
	// convert Transaction interface to Payload interface
	var txs []merkletree.Payload
	for _, tx := range b.Txs {
		txs = append(txs, tx)
	}

	tree, err := merkletree.NewTree(txs)
	if err != nil {
		return nil, err
	}

	return tree.MerkleRoot, nil
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

// CalculateHash will calculate the block hash.
func (b *Block) CalculateHash() ([]byte, error) {
	return b.Header.CalculateHash()
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
