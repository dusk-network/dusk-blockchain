package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

//RandomBlock returns a random block for testing
func RandomBlock(t *testing.T) *block.Block {
	b := &block.Block{
		Header: RandomHeader(t),
		Txs:    RandomSliceOfTxs(t),
	}
	err := b.SetHash()
	assert.Nil(t, err)
	return b
}

// TwoLinkedBlocks returns two blocks that are linked via their headers
func TwoLinkedBlocks(t *testing.T) (*block.Block, *block.Block) {
	blk0 := &block.Block{
		Header: RandomHeader(t),
		Txs:    RandomSliceOfTxs(t),
	}
	err := blk0.SetHash()
	assert.Nil(t, err)

	blk1 := &block.Block{
		Header: RandomHeader(t),
		Txs:    RandomSliceOfTxs(t),
	}

	blk1.Header.PrevBlock = blk0.Header.Hash
	blk1.Header.Height = blk0.Header.Height + 1
	blk1.Header.Timestamp = blk0.Header.Timestamp + 100
	err = blk1.SetRoot()
	assert.Nil(t, err)
	err = blk1.SetHash()
	assert.Nil(t, err)

	return blk0, blk1
}

//RandomCertificate returns a random block certificate  for testing
func RandomCertificate(t *testing.T) *block.Certificate {
	c := &block.Certificate{
		BRBatchedSig: RandomSlice(t, 33),
		BRStep:       20,

		SRBatchedSig: RandomSlice(t, 33),
		SRStep:       12,
	}

	for i := 0; i < 10; i++ {
		c.BRPubKeys = append(c.BRPubKeys, RandomSlice(t, 33))
		c.SRPubKeys = append(c.SRPubKeys, RandomSlice(t, 33))
	}

	err := c.SetHash()
	assert.Nil(t, err)
	return c
}

// RandomHeader returns a random header for testing
func RandomHeader(t *testing.T) *block.Header {

	h := &block.Header{
		Version:   0,
		Timestamp: 2000,
		Height:    200,

		PrevBlock: RandomSlice(t, 32),
		Seed:      RandomSlice(t, 33),
		TxRoot:    RandomSlice(t, 32),

		CertHash: RandomSlice(t, 32),
	}

	return h
}
