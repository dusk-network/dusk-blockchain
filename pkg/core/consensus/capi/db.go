package capi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/tidwall/buntdb"
)

//nolint
var (
	ProvisionersPrefix = "provisioner"
	RoundInfoPrefix    = "roundinfo"
	EventQueuePrefix   = "eventqueue"
	buntStoreInstance  *BuntStore
	dbMu               sync.RWMutex
)

type Level int

const (
	Low    Level = -1
	Medium Level = 0
	High   Level = 1
)

// BuntStore provides access to BuntDB
type BuntStore struct {
	// db is the handle to db
	db *buntdb.DB
	// The path to the BuntDB file
	path string
}

func GetBuntStoreInstance() *BuntStore {
	if buntStoreInstance == nil {
		panic("BuntStore instance is nil")
	}
	return buntStoreInstance
}

func SetBuntStoreInstance(store *BuntStore) {
	buntStoreInstance = store
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
		db.Close()
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
		db.Close()
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

// GetKey will get a composed key
func GetKey(name string, value interface{}) string {
	return fmt.Sprintf("%s:%v", name, value)
}

// FetchProvisioners will get the Provisioners from db
func (b *BuntStore) FetchProvisioners(height uint64) (*user.Provisioners, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	var provisioners user.Provisioners
	err := b.db.View(func(t *buntdb.Tx) error {
		key := GetKey(ProvisionersPrefix, height)
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

// StoreProvisioners will store the Provisioners into db
func (b *BuntStore) StoreProvisioners(provisioners *user.Provisioners, height uint64) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	err := b.db.Update(func(tx *buntdb.Tx) error {

		var init []byte
		buf := bytes.NewBuffer(init)

		err := user.MarshalProvisioners(buf, provisioners)
		if err != nil {
			return err
		}

		key := GetKey(ProvisionersPrefix, height)
		expirationTime := cfg.Get().API.ExpirationTime
		_, _, err = tx.Set(key, buf.String(), &buntdb.SetOptions{Expires: true, TTL: time.Duration(expirationTime) * time.Second})
		return err
	})
	return err
}

// FetchRoundInfo will get the RoundInfoJSON info from db
func (b *BuntStore) FetchRoundInfo(round uint64, stepBegin, stepEnd uint8) ([]RoundInfoJSON, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	var targetJSONList []RoundInfoJSON
	err := b.db.View(func(t *buntdb.Tx) error {
		var err error
		for i := int(stepBegin); i < int(stepEnd); i++ {
			var targetJSON RoundInfoJSON
			key := GetKey(RoundInfoPrefix, fmt.Sprintf("%d:%d", round, i))
			eventQueueJSONStr, err := t.Get(key)
			if err != nil {
				break
			}

			buf := new(bytes.Buffer)
			_, err = buf.WriteString(eventQueueJSONStr)
			if err != nil {
				return err
			}

			err = json.Unmarshal(buf.Bytes(), &targetJSON)
			if err != nil {
				return err
			}

			targetJSONList = append(targetJSONList, targetJSON)
		}
		return err
	})
	return targetJSONList, err
}

// StoreRoundInfo will store the round info into db
func (b *BuntStore) StoreRoundInfo(round uint64, step uint8, methodName, name string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	err := b.db.Update(func(tx *buntdb.Tx) error {
		eventQueueKey := GetKey(RoundInfoPrefix, fmt.Sprintf("%d:%d", round, step))

		eventQueueJSON := RoundInfoJSON{
			Step:      step,
			UpdatedAt: time.Now(),
			Method:    methodName,
			Name:      name,
		}

		eventQueueByteArr, err := json.Marshal(eventQueueJSON)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(eventQueueKey, string(eventQueueByteArr), &buntdb.SetOptions{Expires: true, TTL: time.Duration(cfg.Get().API.ExpirationTime) * time.Second})
		return err
	})
	return err
}

// FetchEventQueue will store the EventQueueJSON info into db
func (b *BuntStore) FetchEventQueue(round uint64, stepBegin, stepEnd uint8) ([]EventQueueJSON, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	var eventQueueJSONList []EventQueueJSON
	err := b.db.View(func(t *buntdb.Tx) error {
		var err error
		for i := int(stepBegin); i < int(stepEnd); i++ {
			var targetJSON EventQueueJSON
			key := GetKey(EventQueuePrefix, fmt.Sprintf("%d:%d", round, i))
			eventQueueJSONStr, err := t.Get(key)
			if err != nil {
				break
			}

			buf := new(bytes.Buffer)
			_, err = buf.WriteString(eventQueueJSONStr)
			if err != nil {
				return err
			}

			err = json.Unmarshal(buf.Bytes(), &targetJSON)

			eventQueueJSONList = append(eventQueueJSONList, targetJSON)
		}
		return err
	})
	return eventQueueJSONList, err
}

// StoreEventQueue will store the round info into db
func (b *BuntStore) StoreEventQueue(round uint64, step uint8, m message.Message) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	err := b.db.Update(func(tx *buntdb.Tx) error {
		eventQueueKey := GetKey(EventQueuePrefix, fmt.Sprintf("%d:%d", round, step))

		eventQueueJSON := EventQueueJSON{
			Round:     round,
			Step:      step,
			Message:   m,
			UpdatedAt: time.Now(),
		}

		eventQueueByteArr, err := json.Marshal(eventQueueJSON)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(eventQueueKey, string(eventQueueByteArr), &buntdb.SetOptions{Expires: true, TTL: time.Duration(cfg.Get().API.ExpirationTime) * time.Second})
		return err
	})
	return err
}
