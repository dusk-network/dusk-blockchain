package responding_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-wallet/block"
)

// Test the behaviour of the data broker, when it receives a GetData message.
func TestSendData(t *testing.T) {
	_, db := lite.CreateDBConnection()
	defer db.Close()

	// Generate 5 blocks and store them in the db, and save the hashes for later checking.
	hashes, blocks := generateBlocks(t, 5)
	if err := storeBlocks(db, blocks); err != nil {
		t.Fatal(err)
	}

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
		if topic != topics.Block {
			t.Fatalf("unexpected topic %s, expected Block", topic)
		}

		// Decode block
		blk := block.NewBlock()
		if err := message.UnmarshalBlock(buf, blk); err != nil {
			t.Fatal(err)
		}

		recvBlocks = append(recvBlocks, blk)
	}

	// Check that block hashes match up with those we generated
	for i, blk := range recvBlocks {
		if !bytes.Equal(hashes[i], blk.Header.Hash) {
			t.Fatal("received block has mismatched hash")
		}
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
