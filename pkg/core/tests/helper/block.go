// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package helper

import (
	"testing"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/stretchr/testify/assert"
)

// RandomBlock returns a random block for testing.
// For `height` see also helper.RandomHeader.
// For txBatchCount see also helper.RandomSliceOfTxs.
func RandomBlock(height uint64, txBatchCount uint16) *block.Block {
	b := &block.Block{
		Header: RandomHeader(height),
		Txs:    transactions.RandContractCalls(int(txBatchCount), 0, true),
	}

	hash, err := b.CalculateHash()
	if err != nil {
		panic(err)
	}

	b.Header.Hash = hash

	return b
}

// TwoLinkedBlocks returns two blocks that are linked via their headers.
func TwoLinkedBlocks(t *testing.T) (*block.Block, *block.Block) {
	blk0 := &block.Block{
		Header: RandomHeader(200),
		Txs:    transactions.RandContractCalls(19, 0, true),
	}

	hash, err := blk0.CalculateHash()
	assert.Nil(t, err)

	blk0.Header.Hash = hash

	blk1 := &block.Block{
		Header: RandomHeader(200),
		Txs:    transactions.RandContractCalls(19, 0, true),
	}

	blk1.Header.PrevBlockHash = blk0.Header.Hash
	blk1.Header.Height = blk0.Header.Height + 1
	blk1.Header.Timestamp = blk0.Header.Timestamp + 100

	hash, err = blk1.CalculateHash()
	assert.Nil(t, err)

	blk1.Header.Hash = hash
	return blk0, blk1
}

// RandomCertificate returns a random block certificate for testing.
func RandomCertificate() *block.Certificate {
	return block.EmptyCertificate()
}

func binary(val byte, size int) []byte {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = val
	}

	return buf
}

// RandomHeader returns a random header for testing. `height` randomness is up
// to the caller. A global atomic counter per pkg can handle it.
func RandomHeader(height uint64) *block.Header {
	return &block.Header{
		Version:   0,
		Height:    height,
		Timestamp: 11112222,
		GasLimit:  123456,

		PrevBlockHash:      binary(10, 32),
		Seed:               binary(11, 32),
		GeneratorBlsPubkey: binary(12, 96),
		StateHash:          binary(13, 32),

		Certificate: RandomCertificate(),
	}
}

// RandomBLSSignature returns a valid BLS Signature of a bogus message.
func RandomBLSSignature() []byte {
	msg := "this is a test"
	keys := key.NewRandKeys()

	sig, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, []byte(msg))
	if err != nil {
		panic(err)
	}

	return sig
}
