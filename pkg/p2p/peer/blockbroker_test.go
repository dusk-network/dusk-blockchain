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

func TestSendBlocks(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	db, err := heavy.NewDatabase(cfg.Get().Database.Dir, protocol.TestNet, false)
	if err != nil {
		t.Fatal(err)
	}

	// Generate 5 blocks and store them in the db, and save the hashes for later checking.
	var hashes [][]byte
	for i := 0; i < 5; i++ {
		blk := helper.RandomBlock(t, uint64(i), 2)
		hashes = append(hashes, blk.Header.Hash)
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

	go func() {
		peerReader, err := helper.StartPeerReader(eb, cs, "3000")
		if err != nil {
			t.Fatal(err)
		}

		peerReader.ReadLoop()
	}()

	time.Sleep(100 * time.Millisecond)

	// Make a GetBlocks, with the genesis block as the locator.
	getBlocks := &peermsg.GetBlocks{}
	getBlocks.Locators = append(getBlocks.Locators, hashes[0])
	getBlocks.Target = make([]byte, 32)

	buf := new(bytes.Buffer)
	if err := getBlocks.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg, err := wire.AddTopic(buf, topics.GetBlocks)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := g.Process(msg)
	if err != nil {
		t.Fatal(err)
	}

	// Connect to the peer and write the message to them
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Write(encoded.Bytes()); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(conn)

	// We should receive 4 new blocks from the peer
	var blocks []*block.Block
	for i := 0; i < 4; i++ {
		bs, err := r.ReadBytes(0x00)
		if err != nil {
			t.Fatal(err)
		}

		decoded := processing.Decode(bs)

		// Remove magic bytes
		if _, err := decoded.Read(make([]byte, 4)); err != nil {
			t.Fatal(err)
		}

		var topicBytes [15]byte
		if _, err := decoded.Read(topicBytes[:]); err != nil {
			t.Fatal(err)
		}

		topic := topics.ByteArrayToTopic(topicBytes)
		if topic != topics.Block {
			t.Fatalf("unexpected topic %s, expected Block", topic)
		}

		blk := block.NewBlock()
		if err := blk.Decode(decoded); err != nil {
			t.Fatal(err)
		}

		blocks = append(blocks, blk)
	}

	for i, blk := range blocks {
		if !bytes.Equal(hashes[i+1], blk.Header.Hash) {
			t.Fatal("received block has mismatched hash")
		}
	}
}
