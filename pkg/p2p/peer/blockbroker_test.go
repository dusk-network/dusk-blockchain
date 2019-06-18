package peer_test

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func mockConfig(t *testing.T) func() {
	storeDir, err := ioutil.TempDir(os.TempDir(), "peer_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	r := cfg.Registry{}
	r.Database.Dir = storeDir
	r.Database.Driver = heavy.DriverName
	r.General.Network = "testnet"
	cfg.Mock(&r)

	return func() {
		os.RemoveAll(storeDir)
	}
}

// Test the behaviour of the block broker
func TestSendBlocks(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	// Set up db
	// TODO: use a mock for this instead
	db, err := heavy.NewDatabase(cfg.Get().Database.Dir, protocol.TestNet, false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Generate 5 blocks and store them in the db, and save the hashes for later checking.
	hashes := generateBlocks(t, 5, db)

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

	time.Sleep(2 * time.Second)

	// Make a GetBlocks, with the genesis block as the locator.
	msg := createGetBlocksBuffer(hashes[0], hashes[4], g)

	if _, err := client.Write(msg.Bytes()); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(client)

	// We should receive 3 new blocks from the peer
	var blocks []*block.Block
	for i := 0; i < 3; i++ {
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

		blocks = append(blocks, blk)
	}

	// Check that block hashes match up with those we generated
	for i, blk := range blocks {
		if !bytes.Equal(hashes[i+1], blk.Header.Hash) {
			t.Fatal("received block has mismatched hash")
		}
	}
}

func generateBlocks(t *testing.T, amount int, db database.DB) [][]byte {
	var hashes [][]byte
	for i := 0; i < amount; i++ {
		blk := helper.RandomBlock(t, uint64(i), 2)
		hashes = append(hashes, blk.Header.Hash)
		err := db.Update(func(t database.Transaction) error {
			return t.StoreBlock(blk)
		})

		if err != nil {
			panic(err)
		}
	}

	return hashes
}

func createGetBlocksBuffer(locator, target []byte, g *processing.Gossip) *bytes.Buffer {
	getBlocks := &peermsg.GetBlocks{}
	getBlocks.Locators = append(getBlocks.Locators, locator)
	getBlocks.Target = target

	buf := new(bytes.Buffer)
	if err := getBlocks.Encode(buf); err != nil {
		panic(err)
	}

	msg, err := wire.AddTopic(buf, topics.GetBlocks)
	if err != nil {
		panic(err)
	}

	encoded, err := g.Process(msg)
	if err != nil {
		panic(err)
	}

	return encoded
}
