package database

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DB encapsulates a leveldb.DB storage
type DB struct {
	storage *leveldb.DB
}

var (
	txRecordPrefix = []byte{0x00}

	writeOptions = &opt.WriteOptions{NoWriteMerge: false, Sync: true}
)

// New creates an instance of DB
func New(path string) (*DB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("wallet cannot be used without database %s", err.Error())
	}
	return &DB{storage: db}, nil
}

// Put inserts a key and a value in the storage
func (db *DB) Put(key, value []byte) error {
	return db.storage.Put(key, value, nil)
}

// Get a value using a key
func (db *DB) Get(key []byte) ([]byte, error) {
	return db.storage.Get(key, nil)
}

// Delete an entry identified by the specified key
func (db *DB) Delete(key []byte) error {
	return db.storage.Delete(key, nil)
}

// Close the storage
func (db *DB) Close() error {
	return db.storage.Close()
}

// FetchTxRecords transaction records
func (db *DB) FetchTxRecords() ([]txrecords.TxRecord, error) {
	records := make([]txrecords.TxRecord, 0)
	iter := db.storage.NewIterator(util.BytesPrefix(txRecordPrefix), nil)
	defer iter.Release()

	for iter.Next() {
		// record is the key without the prefix
		val := iter.Key()[1:]

		bs := make([]byte, len(val))
		copy(bs[:], val)

		txRecord := txrecords.TxRecord{}

		if err := txrecords.Decode(bytes.NewBuffer(bs), &txRecord); err != nil {
			return nil, err
		}

		records = append(records, txRecord)
	}

	err := iter.Error()
	return records, err
}

// PutTxRecord saves the transaction record on the DB
func (db *DB) PutTxRecord(tx transactions.ContractCall, height uint64, direction txrecords.Direction, privView *key.PrivateView) error {
	// Schema
	//
	// key: txRecordPrefix + record
	// value: 0
	buf := new(bytes.Buffer)
	txRecord := txrecords.New(tx, height, direction, privView)
	if err := txrecords.Encode(buf, txRecord); err != nil {
		return err
	}

	key := make([]byte, 0)
	key = append(key, txRecordPrefix...)
	key = append(key, buf.Bytes()...)

	value := make([]byte, 1)
	return db.Put(key, value)
}

// Clear all information from the database.
func (db *DB) Clear() error {
	iter := db.storage.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		if err := db.Delete(iter.Key()); err != nil {
			return err
		}
	}

	return iter.Error()
}
