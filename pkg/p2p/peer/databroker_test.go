package peer_test

import (
	"bufio"
	"bytes"
	"net"
	"testing"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// Test the behaviour of the data broker
func TestSendData(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	// Set up db
	// TODO: use a mock for this instead
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		t.Fatal(err)
	}
	defer drvr.Close()

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.TestNet, false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Generate 5 blocks and store them in the db, and save the hashes for later checking.
	hashes, blocks := generateBlocks(t, 5, db)
	for _, blk := range blocks {
		err := db.Update(func(t database.Transaction) error {
			return t.StoreBlock(blk)
		})

		if err != nil {
			t.Fatal(err)
		}
	}

	eb := wire.NewEventBus()
	cs := chainsync.LaunchChainSynchronizer(eb, protocol.TestNet)
	g := processing.NewGossip(protocol.TestNet)
	client, srv := net.Pipe()

	go func() {
		peerReader, err := helper.StartPeerReader(srv, eb, cs)
		if err != nil {
			t.Fatal(err)
		}

		peerReader.ReadLoop()
	}()

	time.Sleep(500 * time.Millisecond)

	// Make a GetData
	msg := createGetDataBuffer(g, hashes...)

	if _, err := client.Write(msg.Bytes()); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(client)

	// We should receive 5 new blocks from the peer
	var recvBlocks []*block.Block
	for i := 0; i < 5; i++ {
		bs, err := r.ReadBytes(0x00)
		if err != nil {
			t.Fatal(err)
		}

		decoded := processing.Decode(bs)

		// Remove magic bytes
		if _, err := decoded.Read(make([]byte, 4)); err != nil {
			t.Fatal(err)
		}

		// Check for correctness of topic
		var topicBytes [15]byte
		if _, err := decoded.Read(topicBytes[:]); err != nil {
			t.Fatal(err)
		}

		topic := topics.ByteArrayToTopic(topicBytes)
		if topic != topics.Block {
			t.Fatalf("unexpected topic %s, expected Block", topic)
		}

		// Decode block from the stream
		blk := block.NewBlock()
		if err := blk.Decode(decoded); err != nil {
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
func createGetDataBuffer(g *processing.Gossip, hashes ...[]byte) *bytes.Buffer {
	inv := &peermsg.Inv{}
	for _, hash := range hashes {
		inv.AddItem(peermsg.InvTypeBlock, hash)
	}

	buf := new(bytes.Buffer)
	if err := inv.Encode(buf); err != nil {
		panic(err)
	}

	msg, err := wire.AddTopic(buf, topics.GetData)
	if err != nil {
		panic(err)
	}

	encoded, err := g.Process(msg)
	if err != nil {
		panic(err)
	}

	return encoded
}
