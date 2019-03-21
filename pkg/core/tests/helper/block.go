package helper

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

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
