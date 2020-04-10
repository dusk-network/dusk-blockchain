package lite

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/utils"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
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
	if err := message.MarshalBlock(buf, b); err != nil {
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

		for i, output := range tx.StandardTx().Outputs {
			value := make([]byte, 8)
			// Only lock the first output, so that change outputs are
			// not affected.
			if i == 0 {
				binary.LittleEndian.PutUint64(value, tx.LockTime()+b.Header.Height)
			}
			t.batch[outputKeyInd][toKey(output.PubKey.P.Bytes())] = value
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

	// Remove expired bid values
	for k := range t.db.storage[bidValuesInd] {
		heightBytes := k[9:]
		height := binary.LittleEndian.Uint64(heightBytes)
		if height < b.Header.Height {
			delete(t.db.storage[bidValuesInd], k)
		}
	}

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
	if err := message.UnmarshalBlock(bytes.NewBuffer(data), b); err != nil {
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
	if err := message.UnmarshalBlock(bytes.NewBuffer(data), b); err != nil {
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
	if err := message.UnmarshalBlock(bytes.NewBuffer(data), b); err != nil {
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
	points := make([]ristretto.Point, 0, numDecoys)
	for key := range t.db.storage[outputKeyInd] {
		// Ignore locked outputs
		unlockHeight := binary.LittleEndian.Uint64(t.db.storage[outputKeyInd][key])
		if unlockHeight != 0 {
			continue
		}

		var p ristretto.Point
		var pBytes [32]byte
		copy(pBytes[:], key[:])
		p.SetBytes(&pBytes)

		points = append(points, p)
		if len(points) == numDecoys {
			break
		}
	}

	return points
}

func (t transaction) FetchOutputExists(destkey []byte) (bool, error) {
	_, exists := t.db.storage[outputKeyInd][toKey(destkey)]
	return exists, nil
}

func (t transaction) FetchOutputUnlockHeight(destkey []byte) (uint64, error) {
	unlockHeight, exists := t.db.storage[outputKeyInd][toKey(destkey)]
	if !exists {
		return 0, errors.New("this output does not exist")
	}

	return binary.LittleEndian.Uint64(unlockHeight), nil
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

func (t *transaction) StoreBidValues(d, k []byte, lockTime uint64) error {
	currentHeight, err := t.FetchCurrentHeight()
	if err != nil {
		return err
	}

	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, lockTime+currentHeight)
	key := append([]byte("bidvalues"), heightBytes...)
	bidKey := toKey(key)
	t.batch[bidValuesInd][bidKey] = append(d, k...)
	return nil
}

func (t *transaction) FetchBidValues() ([]byte, []byte, error) {
	// Get bid values with lowest expiry height
	lowestSeen := uint64(1<<64 - 1)
	var values []byte
	for k, v := range t.db.storage[bidValuesInd] {
		heightBytes := k[9:]
		height := binary.LittleEndian.Uint64(heightBytes)
		if height < lowestSeen {
			lowestSeen = height
			values = v
		}
	}

	return values[0:32], values[32:], nil
}

// FetchBlockHeightSince uses binary search to find a block height
// NB: Duplicates FetchBlockHeightSince heavy driver
func (t transaction) FetchBlockHeightSince(sinceUnixTime int64, offset uint64) (uint64, error) {

	tip, err := t.FetchCurrentHeight()
	if err != nil {
		return 0, err
	}

	n := uint64(math.Min(float64(tip), float64(offset)))

	pos, searchErr := utils.Search(n, func(pos uint64) (bool, error) {
		height := tip - n + pos
		hash, e := t.FetchBlockHashByHeight(height)
		if e != nil {
			return false, e
		}

		header, fetchErr := t.FetchBlockHeader(hash)
		if fetchErr != nil {
			return false, fetchErr
		}

		return header.Timestamp >= sinceUnixTime, nil
	})

	if searchErr != nil {
		return 0, searchErr
	}

	return tip - n + pos, nil

}

func (t transaction) ClearDatabase() error {
	for key := range t.db.storage {
		t.db.storage[key] = make(table)
	}

	return nil
}
