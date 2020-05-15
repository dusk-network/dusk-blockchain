package responding_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
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
	responseChan := make(chan *bytes.Buffer, 100)
	dataBroker := responding.NewDataBroker(db, nil, responseChan)

	// Make a GetData and give it to the dataBroker
	msg := createGetDataBuffer(hashes...)
	if err := dataBroker.SendItems(msg); err != nil {
		t.Fatal(err)
	}

	// We should receive 5 new blocks from the peer
	var recvBlocks []*block.Block
	for i := 0; i < 5; i++ {
		buf := <-responseChan

		// Check for correctness of topic
		topic, _ := topics.Extract(buf)
		assert.Equal(topics.Block, topic)

		// Decode block
		blk := block.NewBlock()
		assert.NoError(message.UnmarshalBlock(buf, blk))

		recvBlocks = append(recvBlocks, blk)
	}

	// Check that block hashes match up with those we generated
	for i, blk := range recvBlocks {
		assert.Equal(blk.Header.Hash, hashes[i])
	}
}

// TODO: probably specify somewhere a choice between block and tx type
func createGetDataBuffer(hashes ...[]byte) *bytes.Buffer {
	inv := &peermsg.Inv{}
	for _, hash := range hashes {
		inv.AddItem(peermsg.InvTypeBlock, hash)
	}

	buf := new(bytes.Buffer)
	if err := inv.Encode(buf); err != nil {
		panic(err)
	}

	return buf
}
