package responding_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	assert "github.com/stretchr/testify/require"
)

// Test the behavior of the block hash broker, upon receiving a GetBlocks message.
func TestAdvertiseBlocks(t *testing.T) {
	assert := assert.New(t)
	// Set up db
	_, db := lite.CreateDBConnection()
	defer func() {
		_ = db.Close()
	}()

	// Generate 5 blocks and store them in the db. Save the hashes for later checking.
	hashes, blocks := generateBlocks(5)
	assert.NoError(storeBlocks(db, blocks))

	// Set up the BlockHashBroker
	responseChan := make(chan *bytes.Buffer, 100)
	blockHashBroker := responding.NewBlockHashBroker(db, responseChan)

	// Make a GetBlocks, with the genesis block as the locator.
	msg := createGetBlocksBuffer(hashes[0])
	assert.NoError(blockHashBroker.AdvertiseMissingBlocks(msg))

	// The BlockHashBroker's response should be put on the responseChan.
	response := <-responseChan

	// Check for correctness of topic
	topic, _ := topics.Extract(response)
	assert.Equal(topics.Inv, topic)

	// Decode inv
	inv := &peermsg.Inv{}
	assert.NoError(inv.Decode(response))

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

func createGetBlocksBuffer(locator []byte) *bytes.Buffer {
	getBlocks := &peermsg.GetBlocks{}
	getBlocks.Locators = append(getBlocks.Locators, locator)

	buf := new(bytes.Buffer)
	if err := getBlocks.Encode(buf); err != nil {
		panic(err)
	}

	return buf
}

func storeBlocks(db database.DB, blocks []*block.Block) error {
	for _, blk := range blocks {
		err := db.Update(func(t database.Transaction) error {
			return t.StoreBlock(blk)
		})

		if err != nil {
			return err
		}
	}

	return nil
}
