package processing_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func mockConfig(t *testing.T) func() {
	storeDir, err := ioutil.TempDir(os.TempDir(), "processing_test")
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

func setupDatabase() (database.Driver, database.DB) {
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		panic(err)
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.TestNet, false)
	if err != nil {
		panic(err)
	}

	return drvr, db
}

// Test the behaviour of the block hash broker, upon receiving a GetBlocks message.
func TestAdvertiseBlocks(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	// Set up db
	// TODO: use a mock for this instead
	drvr, db := setupDatabase()
	defer drvr.Close()
	defer db.Close()

	// Generate 5 blocks and store them in the db. Save the hashes for later checking.
	hashes, blocks := generateBlocks(t, 5)
	if err := storeBlocks(db, blocks); err != nil {
		t.Fatal(err)
	}

	// Set up the BlockHashBroker
	responseChan := make(chan *bytes.Buffer, 100)
	blockHashBroker := processing.NewBlockHashBroker(db, responseChan)

	// Make a GetBlocks, with the genesis block as the locator.
	msg := createGetBlocksBuffer(hashes[0])
	if err := blockHashBroker.AdvertiseMissingBlocks(msg); err != nil {
		t.Fatal(err)
	}

	// The BlockHashBroker's response should be put on the responseChan.
	response := <-responseChan

	// Check for correctness of topic
	topic := extractTopic(response)
	if topic != topics.Inv {
		t.Fatalf("unexpected topic %s, expected Inv", topic)
	}

	// Decode inv
	inv := &peermsg.Inv{}
	if err := inv.Decode(response); err != nil {
		t.Fatal(err)
	}

	// Check that block hashes match up with those we generated
	for i, item := range inv.InvList {
		if !bytes.Equal(hashes[i+1], item.Hash) {
			t.Fatal("received inv vector has mismatched hash")
		}
	}
}

// Generate a set of random blocks, which follow each other up in the chain.
func generateBlocks(t *testing.T, amount int) ([][]byte, []*block.Block) {
	var hashes [][]byte
	var blocks []*block.Block
	for i := 0; i < amount; i++ {
		blk := helper.RandomBlock(t, uint64(i), 2)
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

func extractTopic(r io.Reader) topics.Topic {
	var topicBytes [15]byte
	if _, err := r.Read(topicBytes[:]); err != nil {
		panic(err)
	}

	return topics.ByteArrayToTopic(topicBytes)
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
