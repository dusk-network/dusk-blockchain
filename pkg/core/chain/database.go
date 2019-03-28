package chain

import (
	"bytes"
	"io"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

// Database is a mock database interface until Database is functional
type Database interface {
	getBestBlock() (*block.Block, error)
	getBlockHeaderByHash(hash []byte) (*block.Header, error)
	getTxByHash(hash []byte) (transactions.Transaction, error)
	writeBlockHeader(hdr *block.Header) error
	writeBlock(blk block.Block) error
	writeInput(input *transactions.Input) error
	writeTX(tx transactions.Transaction) error
	hasKeyImage(hash []byte) (bool, error)
}

// writeBlock is called after all of the checks on the block pass
// returns nil, if write to database was successful
func (c *Chain) writeBlock(blk block.Block) error {
	c.PrevBlock = blk
	return nil
}

// hasBlock checks whether the block passed as an
// argument has already been saved into our database
// returns nil, if block does not exist
func (c Chain) checkBlockExists(blk block.Block) error {
	if blk.Equals(&c.PrevBlock) {
		return errors.New("block already saved")
	}
	return nil
}

func (c Chain) checkTxExists(tx transactions.Transaction) error {
	hash, _ := tx.CalculateHash()
	if _, err := c.db.getTxByHash(hash); err != nil {
		return err
	}

	return nil
}

type ldb struct {
	storage *leveldb.DB
	path    string

	// If true, accepts read-only Tx
	readOnly bool
}

// NewDatabase a singleton connection to storage
func NewDatabase(path string, readonly bool) (Database, error) {

	storage, err := leveldb.OpenFile(path, nil)

	// Try to recover if corrupted
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		storage, err = leveldb.RecoverFile(path, nil)
	}

	if _, accessdenied := err.(*os.PathError); accessdenied {
		return nil, errors.New("could not open or create db")
	}

	return &ldb{storage, path, readonly}, nil
}

func (l *ldb) hasKeyImage(keyImage []byte) (bool, error) {
	var prefix = []byte("Input")
	var key = append(prefix, keyImage...)
	return l.storage.Has(key, nil)
}

func (l *ldb) getBlockHeaderByHash(hash []byte) (*block.Header, error) {
	var prefix = []byte("HEADER")
	var key = append(prefix, hash...)

	hdr, err := l.storage.Get(key, nil) // Returns err if value not found
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(hdr)

	blockHeader := &block.Header{}
	err = blockHeader.Decode(reader)
	if err != nil {
		return nil, err
	}
	return blockHeader, nil
}

func (l *ldb) getTxByHash(hash []byte) (transactions.Transaction, error) {
	var prefix = []byte("TX")
	var key = append(prefix, hash...)

	_, err := l.storage.Get(key, nil) // Returns err if value not found
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (l *ldb) writeBlockHeader(hdr *block.Header) error {
	var prefix = []byte("HEADER")
	var key = append(prefix, hdr.Hash...)

	val, err := hdr.Bytes()
	if err != nil {
		return err
	}
	return l.storage.Put(key, val, nil)
}

func (l *ldb) writeBlock(blk block.Block) error {
	// Do not use in production: Not atomic

	// Write Header first
	l.writeBlockHeader(blk.Header)

	// Write TXs
	for _, tx := range blk.Txs {
		err := l.writeTX(tx)
		if err != nil {
			return err
		}
	}
	return nil
}
func (l *ldb) writeInput(input *transactions.Input) error {
	// Write Input
	// This can double up as the KeyImage database
	// Because the key used is the keyImage

	key := append([]byte("Input"), input.KeyImage...)
	val, err := toBytes(input.Encode)
	if err != nil {
		return err
	}
	return l.storage.Put(key, val, nil)
}

func (l *ldb) writeTX(tx transactions.Transaction) error {

	// Write standard fields
	hash, err := tx.CalculateHash()
	if err != nil {
		return err
	}
	standard := tx.StandardTX()

	// Save each input as a whole
	for _, input := range standard.Inputs {
		// Saves input
		err := l.writeInput(input)
		if err != nil {
			return err
		}
	}

	// Save whole tx
	var key = append([]byte("TX"), hash...)
	val, err := toBytes(tx.Encode)
	if err != nil {
		return err
	}
	return l.storage.Put(key, val, nil)

}

func (l *ldb) getBestBlock() (*block.Block, error) {
	return nil, nil
}

func toBytes(f func(io.Writer) error) ([]byte, error) {
	buf := new(bytes.Buffer)

	err := f(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
