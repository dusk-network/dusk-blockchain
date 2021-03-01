// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	assert "github.com/stretchr/testify/require"
)

// Test the behavior of the data broker, when it receives a GetData message.
func TestSendData(t *testing.T) {
	assert := assert.New(t)
	_, db := lite.CreateDBConnection()

	defer func() {
		_ = db.Close()
	}()

	// Generate 5 blocks and store them in the db, and save the hashes for later checking.
	hashes, blocks := generateBlocks(5)
	assert.NoError(storeBlocks(db, blocks))
	// Set up DataBroker
	dataBroker := responding.NewDataBroker(db, nil)

	// Make a GetData and give it to the dataBroker
	msg := createGetData(hashes...)

	bufs, err := dataBroker.MarshalObjects("", msg)
	if err != nil {
		t.Fatal(err)
	}

	// We should receive 5 new blocks from the broker
	recvBlocks := make([]*block.Block, 0, 5)

	for _, buf := range bufs {
		// Check for correctness of topic
		topic, _ := topics.Extract(&buf)
		assert.Equal(topics.Block, topic)

		// Decode block
		blk := block.NewBlock()
		assert.NoError(message.UnmarshalBlock(&buf, blk))

		recvBlocks = append(recvBlocks, blk)
	}

	// Check that block hashes match up with those we generated
	for i, blk := range recvBlocks {
		assert.Equal(blk.Header.Hash, hashes[i])
	}
}

// TODO: probably specify somewhere a choice between block and tx type.
func createGetData(hashes ...[]byte) message.Message {
	inv := &message.Inv{}

	for _, hash := range hashes {
		inv.AddItem(message.InvTypeBlock, hash)
	}

	return message.New(topics.GetData, *inv)
}
