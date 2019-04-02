package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"time"
)

//RandomBlock returns a random block for testing.
// For `height` see also helper.RandomHeader
// Fir txBatchCount see also helper.RandomSliceOfTxs
func RandomBlock(t *testing.T, height uint64, txBatchCount uint16) *block.Block {
	b := &block.Block{
		Header: RandomHeader(t, height),
		Txs:    RandomSliceOfTxs(t, txBatchCount),
	}
	err := b.SetHash()

	assert.Nil(t, err)
	return b
}

// TwoLinkedBlocks returns two blocks that are linked via their headers
func TwoLinkedBlocks(t *testing.T) (*block.Block, *block.Block) {
	blk0 := &block.Block{
		Header: RandomHeader(t, 200),
		Txs:    RandomSliceOfTxs(t, 20),
	}
	err := blk0.SetHash()
	assert.Nil(t, err)

	blk1 := &block.Block{
		Header: RandomHeader(t, 200),
		Txs:    RandomSliceOfTxs(t, 20),
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

// RandomHeader returns a random header for testing. `height` randomness is up
// to the caller. A global atomic counter per pkg can handle it
func RandomHeader(t *testing.T, height uint64) *block.Header {

	h := &block.Header{
		Version:   0,
		Timestamp: time.Now().Unix(),
		Height:    height,

		PrevBlock: RandomSlice(t, 32),
		Seed:      RandomSlice(t, 33),
		TxRoot:    RandomSlice(t, 32),

		CertHash: RandomSlice(t, 32),
	}

	return h
}
