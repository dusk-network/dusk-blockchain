package block

import (
	"bytes"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-crypto/merkletree"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/protobuf/proto"
)

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header *Header
	Txs    []rusk.ContractCall
}

type SHA3Payload struct {
	hash []byte
}

func newSHA3Payload(tx rusk.ContractCall) (SHA3Payload, error) {
	out, err := proto.Marshal(tx)
	if err != nil {
		return SHA3Payload{}, err
	}

	h, herr := hash.Sha3256(out)
	if herr != nil {
		return SHA3Payload{}, err
	}
	return SHA3Payload{h}, nil
}

func (s SHA3Payload) CalculateHash() ([]byte, error) {
	return s.hash, nil
}

func (s SHA3Payload) Equals(other interface{}) bool {
	sother, ok := other.(SHA3Payload)
	if !ok {
		cc, right := other.(rusk.ContractCall)
		if !right {
			return false
		}
		var err error
		sother, err = newSHA3Payload(&cc)
		if err != nil {
			return false
		}
	}
	return bytes.Equal(s.hash, sother.hash)
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
		stx, err := newSHA3Payload(&tx)
		if err != nil {
			return nil, err
		}
		txs = append(txs, stx)
	}

	tree, err := merkletree.NewTree(txs)
	if err != nil {
		return nil, err
	}

	return tree.MerkleRoot, nil
}

// AddTx will add a transaction to the block.
func (b *Block) AddTx(tx rusk.ContractCall) {
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
		tx, _ := newSHA3Payload(&b.Txs[i])
		otherTx, _ := newSHA3Payload(&other.Txs[i])

		if !tx.Equals(otherTx) {
			return false
		}
	}

	return true
}
