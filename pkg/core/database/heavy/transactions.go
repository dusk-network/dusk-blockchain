package heavy

import (
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

var (
	// Explicitly show we don't want fsync on each batch write
	fsyncEnabled = false

	// TODO:
	optionNoWriteMerge = false
	writeOptions       = &opt.WriteOptions{NoWriteMerge: optionNoWriteMerge, Sync: fsyncEnabled}
)

// A writable transaction would Put/Delete into leveldb.Batch only
// to achieve atomicity on changing blockchain state
type Tx struct {
	writable bool
	db       *DB

	// Put/Delete must be applied into the batch only
	batch  *leveldb.Batch
	closed bool
}

// GetBlockHeaderByHash gives the block header from the hash
func (t Tx) GetBlockHeaderByHash(hash []byte) (*block.Header, error) {

	// only a dummy get
	h := &block.Header{}
	value, err := t.db.storage.Get(hash, nil)

	if err != nil {
		return nil, err
	}

	h.Height = binary.LittleEndian.Uint64(value)
	return h, err
}

// WriteHeader writes dummy data
func (t Tx) WriteHeader(header *block.Header) error {

	//  only a dummy put
	key := header.Hash
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value[0:], header.Height)
	t.batch.Put(key, value)

	return nil
}

// Commit writes a batch to LevelDB storage
func (t *Tx) Commit() error {
	if !t.writable {
		return errors.New("read-only transaction cannot commit changes")
	}
	// To prevent accidentally commit more than once
	if t.closed {
		return errors.New("already closed transaction cannot commit changes")
	}
	t.closed = true
	return t.db.storage.Write(t.batch, writeOptions)
}

func (t Tx) Rollback() error {
	// LevelDB is supposed to be ACID-complient
	return nil
}

func (t Tx) BlockExists(hdr *block.Header) bool {
	return false
}
