package database

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"os"
)

var (
	ErrOpenLevelDb = errors.New("Could not open or create db.\n" +
		"Please note that LevelDB is not designed for multi-process access.\n")
)

type Database interface {
	Has(key []byte) (bool, error)
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close() error
}

type LDB struct {
	db *leveldb.DB
}

func NewDatabase(path string) (*BlockchainDB, error) {
	// Open Dusk blockchain db or create (if it not alresdy exists)
	db, err := leveldb.OpenFile(path, nil)

	// Try to recover if corrupted
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(path, nil)
	}

	if _, accessdenied := err.(*os.PathError); accessdenied {
		return nil, ErrOpenLevelDb
	}

	return &BlockchainDB{&LDB{db}, path}, nil
}

func (l *LDB) Has(key []byte) (bool, error) {
	return l.db.Has(key, nil)
}

func (l *LDB) Put(key []byte, value []byte) error {
	return l.db.Put(key, value, nil)
}

func (l *LDB) Get(key []byte) ([]byte, error) {
	return l.db.Get(key, nil)
}

func (l *LDB) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

func (l *LDB) Close() error {
	return l.db.Close()
}

func Uint32ToBytes(h uint32) []byte {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, h)
	return a
}

func Uint64ToBytes(h uint64) []byte {
	a := make([]byte, 8)
	binary.BigEndian.PutUint64(a, h)
	return a
}
