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
