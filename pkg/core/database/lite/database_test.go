package lite

import (
	"encoding/binary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"os"
	"testing"
)

func TestReadWriteDatabase(t *testing.T) {

	path := "temp-sqlite3.db"
	dir, _ := os.Getwd()
	os.Remove(dir + "/" + path)

	db, err := NewDatabase(path, false)

	if err != nil {
		t.Fatal("Failed on initiating new Database")
	}

	// Dummy header data to verify read/write capabilities
	h := &block.Header{}
	h.Height = 1111
	h.Hash = make([]byte, 4)
	binary.LittleEndian.PutUint16(h.Hash[0:], 0x2222)

	// read-write Tx
	err = db.Update(func(tx database.Tx) error {
		err := tx.WriteHeader(h)
		return err
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	var retrievedHeight uint64
	// read-only Tx
	err = db.View(func(tx database.Tx) error {
		header, err := tx.GetBlockHeaderByHash(h.Hash)
		retrievedHeight = header.Height
		return err
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	if retrievedHeight != h.Height {
		t.Fatal("Wrong read/write tx")
	}

	// delete temp db
	os.Remove(dir + "/" + path)
}
