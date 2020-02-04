package helper

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/stretchr/testify/assert"
)

// RandomBlock returns a random block for testing.
// For `height` see also helper.RandomHeader
// For txBatchCount see also helper.RandomSliceOfTxs
func RandomBlock(t *testing.T, height uint64, txBatchCount uint16) *block.Block {
	b := &block.Block{
		Header: RandomHeader(t, height),
		Txs:    RandomSliceOfTxs(t, txBatchCount),
	}
	hash, err := b.CalculateHash()
	assert.NoError(t, err)
	b.Header.Hash = hash
	root, err := b.CalculateRoot()
	assert.NoError(t, err)
	b.Header.TxRoot = root
	return b
}

// TwoLinkedBlocks returns two blocks that are linked via their headers
func TwoLinkedBlocks(t *testing.T) (*block.Block, *block.Block) {
	blk0 := &block.Block{
		Header: RandomHeader(t, 200),
		Txs:    RandomSliceOfTxs(t, 20),
	}
	hash, err := blk0.CalculateHash()
	assert.Nil(t, err)
	blk0.Header.Hash = hash

	blk1 := &block.Block{
		Header: RandomHeader(t, 200),
		Txs:    RandomSliceOfTxs(t, 20),
	}

	blk1.Header.PrevBlockHash = blk0.Header.Hash
	blk1.Header.Height = blk0.Header.Height + 1
	blk1.Header.Timestamp = blk0.Header.Timestamp + 100
	root, err := blk1.CalculateRoot()
	assert.Nil(t, err)
	blk1.Header.TxRoot = root
	hash, err = blk1.CalculateHash()
	assert.Nil(t, err)
	blk1.Header.Hash = hash

	return blk0, blk1
}

// RandomCertificate returns a random block certificate for testing
func RandomCertificate(t *testing.T) *block.Certificate {
	return block.EmptyCertificate()
}

// RandomHeader returns a random header for testing. `height` randomness is up
// to the caller. A global atomic counter per pkg can handle it
func RandomHeader(t *testing.T, height uint64) *block.Header {

	h := &block.Header{
		Version:   0,
		Height:    height,
		Timestamp: time.Now().Unix(),

		PrevBlockHash: RandomSlice(t, 32),
		Seed:          RandomSlice(t, 33),
		TxRoot:        RandomSlice(t, 32),

		Certificate: RandomCertificate(t),
	}

	return h
}
