package lite

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/utils"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/transactions"
)

type transaction struct {
	writable bool
	db       *DB
	batch    memdb
}

// NB: More optimal data structure can be used to speed up fetching. E.g instead
// map lookup operation on block per height, one can utilize a height as index
// in a slice.
// NB: A single slice of all blocks to be used to avoid all duplications
func (t *transaction) StoreBlock(b *block.Block) error {

	if !t.writable {
		return errors.New("read-only transaction")
	}

	if len(t.batch) == 0 {
		return errors.New("empty batch")
	}

	// Map header.Hash to block.Block
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, b); err != nil {
		return err
	}

	blockBytes := buf.Bytes()

	t.batch[blocksInd][toKey(b.Header.Hash)] = blockBytes

	// Map txId to transactions.Transaction
	for i, tx := range b.Txs {

		txID, err := tx.CalculateHash()
		if err != nil {
			return err
		}

		if len(txID) == 0 {
			return fmt.Errorf("empty chain tx id")
		}

		data, err := utils.EncodeBlockTx(tx, uint32(i))
		if err != nil {
			return err
		}

		t.batch[txsInd][toKey(txID)] = data
		t.batch[txHashInd][toKey(txID)] = b.Header.Hash

		// Map KeyImage to Transaction
		for _, input := range tx.StandardTx().Inputs {
			t.batch[keyImagesInd][toKey(input.KeyImage.Bytes())] = txID
		}
	}

	// Map height to buffer bytes
	buf = new(bytes.Buffer)

	// Append height value
	if err := utils.WriteUint64(buf, b.Header.Height); err != nil {
		return err
	}
	t.batch[heightInd][toKey(buf.Bytes())] = blockBytes

	// Map stateKey to chain state (tip)
	t.batch[stateInd][toKey(stateKey)] = b.Header.Hash

	return nil
}

// Commit writes a batch to LevelDB storage. See also fsyncEnabled variable
func (t *transaction) Commit() error {
	if !t.writable {
		return errors.New("read-only transaction cannot commit changes")
	}

	/// commit changes
	for i := range t.db.storage {
		for k, v := range t.batch[i] {
			t.db.storage[i][k] = v
		}
	}

	return nil
}

func (t transaction) FetchBlockExists(hash []byte) (bool, error) {

	if _, ok := t.db.storage[blocksInd][toKey(hash)]; !ok {
		return false, database.ErrBlockNotFound
	}
	return true, nil
}

func (t transaction) FetchBlockHeader(hash []byte) (*block.Header, error) {

	var data []byte
	var exists bool
	if data, exists = t.db.storage[blocksInd][toKey(hash)]; !exists {
		return nil, database.ErrBlockNotFound
	}

	b := block.NewBlock()
	if err := marshalling.UnmarshalBlock(bytes.NewBuffer(data), b); err != nil {
		return nil, err
	}

	return b.Header, nil
}

func (t transaction) FetchBlockTxs(hash []byte) ([]transactions.Transaction, error) {

	var data []byte
	var exists bool
	if data, exists = t.db.storage[blocksInd][toKey(hash)]; !exists {
		return nil, database.ErrBlockNotFound
	}

	b := block.NewBlock()
	if err := marshalling.UnmarshalBlock(bytes.NewBuffer(data), b); err != nil {
		return nil, err
	}

	return b.Txs, nil
}

func (t transaction) FetchBlockHashByHeight(height uint64) ([]byte, error) {

	heightBuf := new(bytes.Buffer)

	// Append height value
	if err := utils.WriteUint64(heightBuf, height); err != nil {
		return nil, err
	}

	var data []byte
	var exists bool
	if data, exists = t.db.storage[heightInd][toKey(heightBuf.Bytes())]; !exists {
		return nil, database.ErrBlockNotFound
	}

	b := block.NewBlock()
	if err := marshalling.UnmarshalBlock(bytes.NewBuffer(data), b); err != nil {
		return nil, err
	}

	return b.Header.Hash, nil
}

func (t transaction) FetchBlockTxByHash(txID []byte) (transactions.Transaction, uint32, []byte, error) {

	var data []byte
	var exists bool
	if data, exists = t.db.storage[txsInd][toKey(txID)]; !exists {
		return nil, math.MaxUint32, nil, database.ErrTxNotFound
	}

	tx, txIndex, err := utils.DecodeBlockTx(data, database.AnyTxType)
	if err != nil {
		return nil, math.MaxUint32, nil, database.ErrTxNotFound
	}

	var hash []byte
	if hash, exists = t.db.storage[txHashInd][toKey(txID)]; !exists {
		return nil, math.MaxUint32, nil, database.ErrTxNotFound
	}

	return tx, txIndex, hash, err
}

func (t transaction) FetchKeyImageExists(keyImage []byte) (bool, []byte, error) {

	var txID []byte
	var exists bool
	if txID, exists = t.db.storage[keyImagesInd][toKey(keyImage)]; !exists {
		return false, nil, database.ErrKeyImageNotFound
	}

	return true, txID, nil
}
func (t transaction) FetchDecoys(numDecoys int) []ristretto.Point {
	return nil
}

func (t transaction) FetchOutputExists(destkey []byte) (bool, error) {
	return false, nil
}

func (t transaction) FetchOutputUnlockHeight(destkey []byte) (uint64, error) {
	return 0, nil
}

func (t transaction) FetchState() (*database.State, error) {

	var hash []byte
	var exists bool
	if hash, exists = t.db.storage[stateInd][toKey(stateKey)]; !exists {
		return nil, database.ErrStateNotFound
	}

	if len(hash) == 0 {
		return nil, database.ErrStateNotFound
	}

	s := &database.State{}
	s.TipHash = hash

	return s, nil
}

func toKey(d []byte) key {
	var k key
	copy(k[:], d)
	return k
}

// Rollback is not used by database layer
func (t transaction) Rollback() error {
	return nil
}

func (t *transaction) Close() {
}

func (t *transaction) FetchBlock(hash []byte) (*block.Block, error) {
	header, err := t.FetchBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	txs, err := t.FetchBlockTxs(hash)
	if err != nil {
		return nil, err
	}

	return &block.Block{
		Header: header,
		Txs:    txs,
	}, nil
}

func (t *transaction) FetchCurrentHeight() (uint64, error) {
	state, err := t.FetchState()
	if err != nil {
		return 0, err
	}

	header, err := t.FetchBlockHeader(state.TipHash)
	if err != nil {
		return 0, err
	}

	return header.Height, nil
}

func (t *transaction) StoreBidValues(d, k []byte) error {
	bidKey := toKey([]byte("bidvalues"))
	t.batch[bidValuesInd][bidKey] = append(d, k...)
	return nil
}

func (t *transaction) FetchBidValues() ([]byte, []byte, error) {
	bidKey := toKey([]byte("bidvalues"))
	return t.db.storage[bidValuesInd][bidKey][0:32], t.db.storage[bidValuesInd][bidKey][32:64], nil
}
