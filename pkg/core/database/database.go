package database

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
)

var (
	errOpenLevelDb = errors.New("Could not open or create db.\n" +
		"Please note that LevelDB is not designed for multi-process access.\n")
	writeOptions = &opt.WriteOptions{NoWriteMerge: true} //TODO: Needed? There's only one thread!
)

type batchValues map[string][]byte

// Database is the interface to interact with the blockchain db
type Database interface {
	Has(key []byte) (bool, error)
	Put(key []byte, value []byte) error
	Write(bvs *batchValues) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator
	Close() error
}

// LDB holds the LevelDB db
type LDB struct {
	db *leveldb.DB
}

// NewDatabase returns a pointer to a newly created or existing LevelDB blockchain db
func NewDatabase(path string) (*BlockchainDB, error) {
	// Open Dusk blockchain db or create (if it not alresdy exists)
	db, err := leveldb.OpenFile(path, nil)

	// Try to recover if corrupted
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(path, nil)
	}

	if _, accessdenied := err.(*os.PathError); accessdenied {
		return nil, errOpenLevelDb
	}

	return &BlockchainDB{&LDB{db}, path}, nil
}

// Has returns true if the db does contains the given key
func (l *LDB) Has(key []byte) (bool, error) {
	return l.db.Has(key, nil)
}

// Put sets the value for the given key. It overwrites any previous value.
func (l *LDB) Put(key []byte, value []byte) error {
	return l.db.Put(key, value, nil)
}

// Write apply the given batch to the db
func (l *LDB) Write(bvs *batchValues) error {
	batch := new(leveldb.Batch)
	for k, v := range *bvs {
		batch.Put([]byte(k[:]), v)
	}
	return l.db.Write(batch, writeOptions)
}

// Get gets the value for the given key
func (l *LDB) Get(key []byte) ([]byte, error) {
	return l.db.Get(key, nil)
}

// Delete deletes the value for the given key
func (l *LDB) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

// NewIterator returns an iterator of range of keys
func (l *LDB) NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator {
	return l.db.NewIterator(slice, ro)
}

// Close closes the db
func (l *LDB) Close() error {
	return l.db.Close()
}

// Uint32ToBytes converts a uint32 to a byte slice
func Uint32ToBytes(h uint32) []byte {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, h)
	return a
}

// Uint64ToBytes converts a uint64 to a byte slice
func Uint64ToBytes(h uint64) []byte {
	a := make([]byte, 8)
	binary.BigEndian.PutUint64(a, h)
	return a
}
