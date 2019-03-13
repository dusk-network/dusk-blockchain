package block

import (
	"errors"
	"io"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header *Header
	Txs    []merkletree.Payload
}

// NewBlock will return an empty Block with an empty BlockHeader.
func NewBlock() *Block {
	return &Block{
		Header: &Header{
			Version: 0x00,
			// CertImage should take up space from creation to
			// ensure proper decoding during block collection.
			CertHash: make([]byte, 32),
		},
	}
}

// NewEmptyBlock will return a fully populated empty block, to be used
// for consensus purposes. Use NewBlock in any other circumstance.
func NewEmptyBlock(prevHeader *Header) (*Block, error) {
	block := &Block{
		Header: &Header{
			Height: prevHeader.Height + 1,
			// CertImage and TxRoot should take up space from creation to
			// ensure proper decoding during block collection.
			CertHash: make([]byte, 32),
			TxRoot:   make([]byte, 32),
		},
	}

	block.SetPrevBlock(prevHeader)

	// Set seed to hash of previous seed
	seedHash, err := hash.Sha3256(prevHeader.Seed)
	if err != nil {
		return nil, err
	}

	// Add one empty byte for encoding purposes
	seedHash = append(seedHash, byte(0))

	block.Header.Seed = seedHash
	block.Header.Timestamp = (time.Now().Unix())
	if err := block.SetHash(); err != nil {
		return nil, err
	}

	return block, nil
}

// SetPrevBlock will set all the previous block hash field from a header.
func (b *Block) SetPrevBlock(prevHeader *Header) {
	b.Header.PrevBlock = prevHeader.Hash
}

// SetRoot will set the block merkle root hash.
func (b *Block) SetRoot() error {
	tree, err := merkletree.NewTree(b.Txs)
	if err != nil {
		return err
	}

	b.Header.TxRoot = tree.MerkleRoot
	return nil
}

// AddTx will add a transaction to the block.
func (b *Block) AddTx(tx *transactions.Stealth) {
	b.Txs = append(b.Txs, tx)
}

// AddCertHash will take a hash from a Certificate and put
// it in the block's CertHash field.
func (b *Block) AddCertHash(cert *Certificate) error {
	if cert.Hash == nil {
		if err := cert.SetHash(); err != nil {
			return err
		}
	}

	b.Header.CertHash = cert.Hash
	return nil
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

// DecodeBlock will decode a Block struct from r and return it.
func DecodeBlock(r io.Reader) (*Block, error) {
	header, err := DecodeHeader(r)
	if err != nil {
		return nil, err
	}

	lTxs, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	txs := make([]merkletree.Payload, lTxs)
	for i := uint64(0); i < lTxs; i++ {
		tx := &transactions.Stealth{}
		tx, err := transactions.DecodeTransaction(r)
		if err != nil {
			return nil, err
		}

		txs[i] = tx
	}

	return &Block{
		Header: header,
		Txs:    txs,
	}, nil
}
