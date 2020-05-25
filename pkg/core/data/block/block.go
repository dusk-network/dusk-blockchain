package block

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-crypto/merkletree"
)

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header *Header                     `json:"header"`
	Txs    []transactions.ContractCall `json:"transactions"`
}

// NewBlock will return an empty Block with an empty BlockHeader.
func NewBlock() *Block {
	return &Block{
		Header: NewHeader(),
	}
}

// Copy returns a deep copy of the Block safe to publish to multiple subscribers
func (b Block) Copy() payload.Safe {
	cpy := Block{}
	cpy.Header = b.Header.Copy()
	if b.Txs != nil {
		cpy.Txs = make([]transactions.ContractCall, len(b.Txs))
		for i, tx := range b.Txs {
			cpy.Txs[i] = tx.Copy().(transactions.ContractCall)
		}
	}

	return cpy
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
		txs = append(txs, tx.(merkletree.Payload))
	}

	tree, err := merkletree.NewTree(txs)
	if err != nil {
		return nil, err
	}

	return tree.MerkleRoot, nil
}

// AddTx will add a transaction to the block.
func (b *Block) AddTx(tx transactions.ContractCall) {
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

	for i := 0; i < len(b.Txs); i++ {
		if !transactions.Equal(b.Txs[i], other.Txs[i]) {
			return false
		}
	}

	return true
}
