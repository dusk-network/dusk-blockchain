// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	assert "github.com/stretchr/testify/require"
)

// Test the behavior of the block hash broker, upon receiving a GetBlocks message.
func TestAdvertiseBlocks(t *testing.T) {
	assert := assert.New(t)
	_, db := lite.CreateDBConnection()

	defer func() {
		_ = db.Close()
	}()

	// Generate 5 blocks and store them in the db. Save the hashes for later checking.
	hashes, blocks := generateBlocks(5)
	assert.NoError(storeBlocks(db, blocks))

	// Set up the BlockHashBroker
	blockHashBroker := responding.NewBlockHashBroker(db)

	// Make a GetBlocks, with the genesis block as the locator.
	msg := createGetBlocks(hashes[0])
	blksBuf, err := blockHashBroker.AdvertiseMissingBlocks("", msg)
	assert.NoError(err)

	// Check for correctness of topic
	topic, _ := topics.Extract(&blksBuf[0])
	assert.Equal(topics.Inv, topic)

	// Decode inv
	inv := &message.Inv{}
	assert.NoError(inv.Decode(&blksBuf[0]))

	// Check that block hashes match up with those we generated
	for i, item := range inv.InvList {
		assert.Equal(item.Hash, hashes[i+1])
	}
}

// Generate a set of random blocks, which follow each other up in the chain.
func generateBlocks(amount int) ([][]byte, []*block.Block) {
	var hashes [][]byte
	var blocks []*block.Block

	for i := 0; i < amount; i++ {
		blk := helper.RandomBlock(uint64(i), 2)
		hashes = append(hashes, blk.Header.Hash)
		blocks = append(blocks, blk)
	}

	return hashes, blocks
}

func createGetBlocks(locator []byte) message.Message {
	getBlocks := &message.GetBlocks{}
	getBlocks.Locators = append(getBlocks.Locators, locator)
	return message.New(topics.GetBlocks, *getBlocks)
}

func storeBlocks(db database.DB, blocks []*block.Block) error {
	for _, blk := range blocks {
		err := db.Update(func(t database.Transaction) error {
			return t.StoreBlock(blk, false)
		})
		if err != nil {
			return err
		}
	}

	return nil
}
