// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package heavy

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/utils"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	// Explicitly show we don't want fsync in sake of faster writes.
	//
	// If false, and the machine crashes, then some recent writes may be lost.
	// Note that if it is just the process that crashes (and the machine does
	// not) then no writes will be lost.
	optionFsyncEnabled = false

	optionNoWriteMerge = false

	optypePut    = 1
	optypeDelete = 0
)

var (
	// writeOptions used by both non-Batch and Batch leveldb.Put.
	writeOptions = &opt.WriteOptions{NoWriteMerge: optionNoWriteMerge, Sync: optionFsyncEnabled}

	// Key values prefixes to provide prefix-based sorting mechanism.
	// Refer to README.md for overview idea.

	// HeaderPrefix is the prefix to identify the Header.
	HeaderPrefix = []byte{0x01}
	// TxPrefix is the prefix to identify Transactions.
	TxPrefix = []byte{0x02}
	// HeightPrefix is the prefix to identify the Height.
	HeightPrefix = []byte{0x03}
	// TxIDPrefix is the prefix to identify the Transaction ID.
	TxIDPrefix = []byte{0x04}
	// TipPrefix is the prefix to identify the hash of the latest blockchain block.
	TipPrefix = []byte{0x05}
	// PersistedPrefix is the prefix to identify the hash of the latest blockchain persisted block.
	PersistedPrefix = []byte{0x06}
	// CandidatePrefix is the prefix to identify Candidate messages.
	CandidatePrefix = []byte{0x07}
)

type transaction struct {
	writable bool
	db       *DB

	// Get/Has/Iterate calls must be applied into the snapshot only.
	snapshot *leveldb.Snapshot

	// Put/Delete calls must be applied into a batch only. Transaction does
	// implement atomicity by a levelDB.Batch constructed during the
	// Transaction.
	batch  *leveldb.Batch
	closed bool
}

// DeleteBlock deletes all records associated with a specified block.
func (t transaction) DeleteBlock(b *block.Block) error {
	return t.modify(optypeDelete, b)
}

// StoreBlock stores the entire block data into storage. No validations are
// applied. Method simply stores the block data into LevelDB storage in an
// atomic way. That said, storage state changes only when Commit() is called on
// Transaction completion.
//
// See also the method body to get an idea of Key-Value data schemas.
//
// It is assumed that StoreBlock would be called much less times than Fetch*
// APIs. Based on that, extra indexing data is put to provide fast-read lookups.
func (t transaction) StoreBlock(b *block.Block, persisted bool) error {
	if len(b.Header.Hash) != block.HeaderHashSize {
		return fmt.Errorf("header hash size is %d but it must be %d", len(b.Header.Hash), block.HeaderHashSize)
	}

	if err := t.modify(optypePut, b); err != nil {
		return err
	}

	// Key = TipPrefix
	// Value = Hash(chain tip)
	//
	// To support fetching  blockchain tip
	t.put(TipPrefix, b.Header.Hash)

	// Key = PersistedPrefix
	// Value = Hash(chain tip)
	//
	// To support fetching blockchain persisted hash
	if persisted {
		t.put(PersistedPrefix, b.Header.Hash)
	}

	return nil
}

func (t *transaction) modify(optype int, b *block.Block) error {
	if t.batch == nil {
		// t.batch is initialized only on a open, read-write transaction
		// (built with transaction.Update()).
		return errors.New("StoreBlock cannot be called on read-only transaction")
	}

	if len(b.Header.Hash) != block.HeaderHashSize {
		return fmt.Errorf("header hash size is %d but it must be %d", len(b.Header.Hash), block.HeaderHashSize)
	}

	// Schema Key = HeaderPrefix + block.header.hash
	//
	// Value = encoded(block.fields)

	blockHeaderFields := new(bytes.Buffer)
	if err := message.MarshalHeader(blockHeaderFields, b.Header); err != nil {
		return err
	}

	key := append(HeaderPrefix, b.Header.Hash...)
	value := blockHeaderFields.Bytes()

	t.op(optype, key, value)

	if uint64(len(b.Txs)) > math.MaxUint32 {
		return errors.New("too many transactions")
	}

	// Put block transaction data. A KV pair per a single transaction is added
	// into the store.
	for i, tx := range b.Txs {
		txID, err := tx.CalculateHash()
		if err != nil {
			return err
		}

		if len(txID) == 0 {
			return fmt.Errorf("empty chain tx id")
		}

		// Schema
		//
		// Key = TxPrefix + block.header.hash + txID
		// Value = index + block.transaction[index]
		//
		// For the retrival of transactions data by block.header.hash
		keys := append(TxPrefix, b.Header.Hash...)
		keys = append(keys, txID...)

		entry, err := utils.EncodeBlockTx(tx, uint32(i))
		if err != nil {
			return err
		}

		t.op(optype, keys, entry)

		// Schema
		//
		// Key = TxIDPrefix + txID
		// Value = block.header.hash
		//
		// For the retrival of a single transaction by TxId
		t.op(optype, append(TxIDPrefix, txID...), b.Header.Hash)
	}

	// Key = HeightPrefix + block.header.height
	// Value = block.header.hash
	//
	// To support fast header lookup by height
	heightBuf := new(bytes.Buffer)

	// Append height value
	if err := utils.WriteUint64(heightBuf, b.Header.Height); err != nil {
		return err
	}

	key = append(HeightPrefix, heightBuf.Bytes()...)

	t.op(optype, key, b.Header.Hash)

	return nil
}

// Commit writes a batch to LevelDB storage. See also fsyncEnabled variable.
func (t *transaction) Commit() error {
	if !t.writable {
		return errors.New("read-only transaction cannot commit changes")
	}

	if t.closed {
		return errors.New("already closed transaction cannot commit changes")
	}

	return t.db.storage.Write(t.batch, writeOptions)
}

// Rollback is not used by database layer.
func (t transaction) Rollback() error {
	t.batch.Reset()
	return nil
}

// Close releases the retrieved snapshot and resets any open batch(s). It must
// be called explicitly when a transaction is run in a unmanaged way.
func (t *transaction) Close() {
	t.snapshot.Release()

	if t.batch != nil {
		t.batch.Reset()
	}

	t.closed = true
}

func (t transaction) FetchBlockExists(hash []byte) (bool, error) {
	key := append(HeaderPrefix, hash...)
	exists, err := t.snapshot.Has(key, nil)

	// goleveldb returns nilIfNotFound
	// see also nilIfNotFound in leveldb/db.go
	if !exists && err == nil {
		// overwrite error message
		err = database.ErrBlockNotFound
	}

	return exists, err
}

func (t transaction) FetchBlockHeader(hash []byte) (*block.Header, error) {
	key := append(HeaderPrefix, hash...)

	value, err := t.snapshot.Get(key, nil)
	if err == leveldb.ErrNotFound {
		// overwrite error message
		err = database.ErrBlockNotFound
	}

	if err != nil {
		return nil, err
	}

	header := block.NewHeader()

	err = message.UnmarshalHeader(bytes.NewBuffer(value), header)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (t transaction) FetchBlockTxs(hashHeader []byte) ([]transactions.ContractCall, error) {
	scanFilter := append(TxPrefix, hashHeader...)
	tempTxs := make(map[uint32]transactions.ContractCall)

	// Read all the transactions that belong to a single block
	// Scan filter = TX_PREFIX + block.header.hash
	iterator := t.snapshot.NewIterator(util.BytesPrefix(scanFilter), nil)
	defer iterator.Release()

	for iterator.Next() {
		value := iterator.Value()

		tx, txIndex, err := utils.DecodeBlockTx(value, database.AnyTxType)
		if err != nil {
			return nil, err
		}

		// If we don't fetch the correct indexes (tx positions), merkle tree
		// changes and as result we've got new block hash
		if _, ok := tempTxs[txIndex]; ok {
			return nil, errors.New("duplicated tx index")
		}

		tempTxs[txIndex] = tx
	}

	// Reorder Tx slice as per retrieved indexes
	resultTxs := make([]transactions.ContractCall, len(tempTxs))
	for k, v := range tempTxs {
		resultTxs[k] = v
	}

	return resultTxs, nil
}

func (t transaction) FetchBlockHashByHeight(height uint64) ([]byte, error) {
	// Get height bytes
	heightBuf := new(bytes.Buffer)
	if err := utils.WriteUint64(heightBuf, height); err != nil {
		return nil, err
	}

	key := append(HeightPrefix, heightBuf.Bytes()...)

	value, err := t.snapshot.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			// overwrite error message
			err = database.ErrBlockNotFound
		}

		return nil, err
	}

	return value, nil
}

func (t transaction) put(key []byte, value []byte) {
	if !t.writable {
		return
	}

	if t.batch != nil {
		t.batch.Put(key, value)
	} else {
		// fail-fast when a writable transaction is not capable of storing data
		log.Panic("leveldb batch is unreachable")
	}
}

func (t transaction) op(opType int, key []byte, value []byte) {
	if !t.writable {
		return
	}

	if t.batch != nil {

		switch opType {
		case optypePut:
			t.batch.Put(key, value)
		case optypeDelete:
			t.batch.Delete(key)
		}

	} else {
		// fail-fast when a writable transaction is not capable of storing data
		log.Panic("leveldb batch is unreachable")
	}
}

func (t transaction) FetchBlockTxByHash(txID []byte) (transactions.ContractCall, uint32, []byte, error) {
	txIndex := uint32(math.MaxUint32)

	// Fetch the block header hash that this Tx belongs to
	key := append(TxIDPrefix, txID...)

	hashHeader, err := t.snapshot.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			// overwrite error message
			err = database.ErrTxNotFound
		}

		return nil, txIndex, nil, err
	}

	// Fetch all the txs that belong to a single block
	// Return only the transaction that is associated to txID
	scanFilter := append(TxPrefix, hashHeader...)

	iterator := t.snapshot.NewIterator(util.BytesPrefix(scanFilter), nil)
	defer iterator.Release()

	for iterator.Next() {
		// Extract TxID from the key to avoid the need of CalculateHash
		reader := bytes.NewReader(iterator.Key())
		fetchedTxID := make([]byte, len(txID))

		// We should always be capable of reading the TxID from the KEY
		n, err := reader.ReadAt(fetchedTxID[:], int64(len(scanFilter)))
		if err != nil || n == 0 {
			return nil, txIndex, nil, fmt.Errorf("malformed transaction data")
		}

		// Check if this is the requested TxID
		if !bytes.Equal(fetchedTxID[:], txID) {
			continue
		}

		// TxID matched. Decode the Tx data
		value := iterator.Value()

		tx, idx, err := utils.DecodeBlockTx(value, database.AnyTxType)
		if err != nil {
			return nil, idx, hashHeader, err
		}

		return tx, idx, hashHeader, nil
	}

	return nil, txIndex, nil, errors.New("block tx is available but fetching it fails")
}

func (t transaction) FetchBlock(hash []byte) (*block.Block, error) {
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

func (t transaction) FetchRegistry() (*database.Registry, error) {
	tipHash, err := t.snapshot.Get(TipPrefix, nil)
	if err == leveldb.ErrNotFound || len(tipHash) == 0 {
		// overwrite error message
		err = database.ErrStateNotFound
	}

	if err != nil {
		return nil, err
	}

	persistedHash, err := t.snapshot.Get(PersistedPrefix, nil)
	if err == leveldb.ErrNotFound || len(persistedHash) == 0 {
		// overwrite error message
		err = database.ErrStateNotFound
	}

	if err != nil {
		return nil, err
	}

	return &database.Registry{
		TipHash:       tipHash,
		PersistedHash: persistedHash,
	}, nil
}

func (t transaction) FetchCurrentHeight() (uint64, error) {
	state, err := t.FetchRegistry()
	if err != nil {
		return 0, err
	}

	header, err := t.FetchBlockHeader(state.TipHash)
	if err != nil {
		return 0, err
	}

	return header.Height, nil
}

// FetchBlockHeightSince uses binary search to find a block height.
func (t transaction) FetchBlockHeightSince(sinceUnixTime int64, offset uint64) (uint64, error) {
	tip, err := t.FetchCurrentHeight()
	if err != nil {
		return 0, err
	}

	n := uint64(math.Min(float64(tip), float64(offset)))

	pos, err := utils.Search(n, func(pos uint64) (bool, error) {
		height := tip - n + pos

		hash, heightErr := t.FetchBlockHashByHeight(height)
		if heightErr != nil {
			return false, heightErr
		}

		header, blockHdrErr := t.FetchBlockHeader(hash)
		if blockHdrErr != nil {
			return false, blockHdrErr
		}

		return header.Timestamp >= sinceUnixTime, nil
	})
	if err != nil {
		return 0, err
	}

	return tip - n + pos, nil
}

func (t transaction) StoreCandidateMessage(cm block.Block) error {
	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, &cm); err != nil {
		return err
	}

	key := append(CandidatePrefix, cm.Header.Hash...)
	t.put(key, buf.Bytes())
	return nil
}

func (t transaction) FetchCandidateMessage(hash []byte) (block.Block, error) {
	key := append(CandidatePrefix, hash...)

	value, err := t.snapshot.Get(key, nil)
	if err != nil {
		return block.Block{}, database.ErrBlockNotFound
	}

	cm := block.NewBlock()
	if err := message.UnmarshalBlock(bytes.NewBuffer(value), cm); err != nil {
		return block.Block{}, err
	}

	return *cm, nil
}

func (t transaction) ClearCandidateMessages() error {
	iter := t.snapshot.NewIterator(util.BytesPrefix(CandidatePrefix), nil)
	defer iter.Release()

	for iter.Next() {
		t.batch.Delete(iter.Key())
	}

	return iter.Error()
}

// ClearDatabase will wipe all of the data currently in the database.
func (t transaction) ClearDatabase() error {
	iter := t.snapshot.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		t.batch.Delete(iter.Key())
	}

	return iter.Error()
}
