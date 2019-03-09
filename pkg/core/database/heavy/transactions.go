package heavy

import (
	"encoding/binary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

var (
	// HEADER is the prefix for a header key
	prefixHeader = []byte{0x01}
)

// Tx provides a root bucket against which all read and writes occur.
type Tx struct {
	closed   bool
	writable bool
	db       *DB
}

// GetBlockHeaderByHash gives the block header from the hash
func (t Tx) GetBlockHeaderByHash(hash []byte) (*block.Header, error) {

	// only a dummy get
	h := &block.Header{}
	value, err := t.db.storage.Get(hash, nil)
	h.Height = binary.LittleEndian.Uint64(value)
	return h, err
}

// WriteHeader writes dummy data
func (t Tx) WriteHeader(header *block.Header) error {

	//  only a dummy put
	key := header.Hash
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value[0:], header.Height)
	err := t.db.storage.Put(key, value, nil)
	return err
}

func (t Tx) Commit() error {
	// LevelDB provides ACID support already
	return nil
}

func (t Tx) Rollback() error {
	// LevelDB provides ACID support already
	return nil
}

func (t Tx) BlockExists(hdr *block.Header) bool {
	return false
}
