package chainsync_test

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"testing"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var port = "8000"

func mockConfig(t *testing.T) func() {
	storeDir, err := ioutil.TempDir(os.TempDir(), "chainsync_test")
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

// Check the behaviour of the ChainSynchronizer when receiving a block, when we
// are sufficiently behind the chain tip.
func TestSynchronizeBehind(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	_ = setUpSynchronizerTest(t)

	// Create a block that is a few rounds in the future
	encodedBlk := createEncodedBlock(t, 5, 20)

	// Connect to peer and write to conn
	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Write(encodedBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	// Read message from conn, we should receive a GetBlocks
	r := bufio.NewReader(conn)
	msg, err := r.ReadBytes(0x00)
	if err != nil {
		t.Fatal(err)
	}

	decoded := processing.Decode(msg)

	// Check topic, which should start at index 4 (magic preceding it)
	var topicBytes [15]byte
	copy(topicBytes[:], decoded.Bytes()[4:19])
	topic := topics.ByteArrayToTopic(topicBytes)
	if topic != topics.GetBlocks {
		t.Fatal("did not receive expected GetBlocks message")
	}
}

// Check the behaviour of the ChainSynchronizer when receiving a block, when we
// are synced with other peers.
func TestSynchronizeSynced(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	eb := setUpSynchronizerTest(t)

	// subscribe to topics.Block
	blockChan := make(chan *bytes.Buffer, 1)
	_ = eb.Subscribe(string(topics.Block), blockChan)

	// Make a block which should follow our genesis block
	encodedBlk := createEncodedBlock(t, 1, 20)

	// Connect to peer and write to conn
	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Write(encodedBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	// The synchronizer should put this block on the blockChan
	<-blockChan
}

func setUpSynchronizerTest(t *testing.T) *wire.EventBus {
	eb := wire.NewEventBus()
	cs := chainsync.LaunchChainSynchronizer(eb, protocol.TestNet)

	// Accept a block at height 0
	blk := randomBlockBuffer(t, 0, 1)
	eb.Publish(string(topics.AcceptedBlock), blk)

	// Set up a peer reader and send it a block that is a few rounds ahead
	go func() {
		peerReader, err := helper.StartPeerReader(eb, cs, port)
		if err != nil {
			t.Fatal(err)
		}
		peerReader.ReadLoop()
	}()

	return eb
}

func randomBlockBuffer(t *testing.T, height uint64, txBatchCount uint16) *bytes.Buffer {
	blk := helper.RandomBlock(t, height, txBatchCount)
	buf := new(bytes.Buffer)
	if err := blk.Encode(buf); err != nil {
		panic(err)
	}

	return buf
}

func createEncodedBlock(t *testing.T, height uint64, txBatchCount uint16) *bytes.Buffer {
	g := processing.NewGossip(protocol.TestNet)
	blk := randomBlockBuffer(t, height, txBatchCount)
	blkWithTopic, err := wire.AddTopic(blk, topics.Block)
	if err != nil {
		t.Fatal(err)
	}

	encodedBlk, err := g.Process(blkWithTopic)
	if err != nil {
		t.Fatal(err)
	}

	return encodedBlk
}
