// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ruskmock

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/tidwall/buntdb"
)

//nolint
var (
	ProvisionersPrefix = "provisioner"
	RoundInfoPrefix    = "roundinfo"
	EventQueuePrefix   = "eventqueue"
	buntStoreInstance  *BuntStore
)

// Level is the database level.
type Level int

//nolint
const (
	Low    Level = -1
	Medium Level = 0
	High   Level = 1
)

// BuntStore provides access to BuntDB.
type BuntStore struct {
	// db is the handle to db
	db *buntdb.DB
	// The path to the BuntDB file
	path string
}

// NewBuntStore takes a file path and returns a connected Raft backend.
func NewBuntStore(path string, durability Level) (*BuntStore, error) {
	// Try to connect
	db, err := buntdb.Open(path)
	if err != nil {
		return nil, err
	}

	// Disable the AutoShrink. Shrinking should only be manually
	// handled following a log compaction.
	var config buntdb.Config
	if err := db.ReadConfig(&config); err != nil {
		_ = db.Close()
		return nil, err
	}

	config.AutoShrinkDisabled = true

	switch durability {
	case Low:
		config.SyncPolicy = buntdb.Never
	case Medium:
		config.SyncPolicy = buntdb.EverySecond
	case High:
		config.SyncPolicy = buntdb.Always
	}

	if err := db.SetConfig(config); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Create the new store
	store := &BuntStore{
		db:   db,
		path: path,
	}
	return store, nil
}

// Close is used to gracefully close the DB connection.
func (b *BuntStore) Close() error {
	return b.db.Close()
}

// GetKey will get a composed key.
func GetKey(name string, value interface{}) string {
	return fmt.Sprintf("%s:%v", name, value)
}

// FetchProvisioners will get the Provisioners from db.
func (b *BuntStore) FetchProvisioners() (*user.Provisioners, error) {
	var provisioners user.Provisioners

	err := b.db.View(func(t *buntdb.Tx) error {
		key := GetKey(ProvisionersPrefix, 0)
		provisionersStr, err := t.Get(key)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		_, err = buf.WriteString(provisionersStr)
		if err != nil {
			return err
		}

		provisioners, err = user.UnmarshalProvisioners(buf)
		return err
	})
	return &provisioners, err
}

// StoreProvisioners will store the Provisioners into db.
func (b *BuntStore) StoreProvisioners(provisioners *user.Provisioners) error {
	// Always reset all keys
	_ = b.Reset()

	err := b.db.Update(func(tx *buntdb.Tx) error {
		buf := bytes.Buffer{}

		err := user.MarshalProvisioners(&buf, provisioners)
		if err != nil {
			return err
		}

		key := GetKey(ProvisionersPrefix, 0)
		_, _, err = tx.Set(key, buf.String(), nil)
		return err
	})
	return err
}

// Reset deletes all keys.
func (b *BuntStore) Reset() error {
	err := b.db.Update(func(tx *buntdb.Tx) error {
		var delkeys []string
		err := tx.AscendKeys("*", func(k, v string) bool {
			delkeys = append(delkeys, k)
			return true // continue
		})
		if err != nil {
			return err
		}

		for _, k := range delkeys {
			if _, err := tx.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
