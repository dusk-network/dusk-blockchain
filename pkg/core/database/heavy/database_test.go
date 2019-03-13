package heavy

import (
	"testing"
)

func TestReadWriteDatabase(t *testing.T) {
	/*
		// on-disk data must be erased before each test run
		storeDir, err := ioutil.TempDir(os.TempDir(), "store_")
		if err != nil {
			t.Fatal(err.Error())
		}
		defer os.RemoveAll(storeDir)

		db, err := NewDatabase(storeDir, false)

		if err != nil {
			t.Fatal("Failed on initiating new Database")
		}

		defer db.Close()

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
			if err != nil {
				return err
			}
			retrievedHeight = header.Height
			return err
		})

		if err != nil {
			t.Fatal(err.Error())
		}

		if retrievedHeight != h.Height {
			t.Fatal("Wrong read/write tx")
		}
	*/
}
