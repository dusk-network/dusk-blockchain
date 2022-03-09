// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package verifiers

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/stretchr/testify/assert"
)

func twoLinkedBlocks(t *testing.T, change int64) (*block.Block, *block.Block) {
	blk0 := &block.Block{
		Header: helper.RandomHeader(200),
		Txs:    transactions.RandContractCalls(19, 0, true),
	}

	hash, err := blk0.CalculateHash()
	assert.Nil(t, err)

	blk0.Header.Hash = hash

	blk1 := &block.Block{
		Header: helper.RandomHeader(200),
		Txs:    transactions.RandContractCalls(19, 0, true),
	}

	blk1.Header.PrevBlockHash = blk0.Header.Hash
	blk1.Header.Height = blk0.Header.Height + 1
	blk1.Header.Timestamp = blk0.Header.Timestamp + change

	root, err := blk1.CalculateRoot()
	assert.Nil(t, err)

	blk1.Header.TxRoot = root

	hash, err = blk1.CalculateHash()
	assert.Nil(t, err)

	blk1.Header.Hash = hash
	return blk0, blk1
}

func TestTimestamp(t *testing.T) {
	a := assert.New(t)

	pb, b := twoLinkedBlocks(t, 0)
	a.NoError(CheckBlockHeader(*pb, *b))

	pb, b = twoLinkedBlocks(t, config.MaxBlockTime)
	a.NoError(CheckBlockHeader(*pb, *b))

	pb, b = twoLinkedBlocks(t, config.MaxBlockTime+1)
	a.NotNil(CheckBlockHeader(*pb, *b))

	pb, b = twoLinkedBlocks(t, -10000)
	a.NotNil(CheckBlockHeader(*pb, *b))
}
