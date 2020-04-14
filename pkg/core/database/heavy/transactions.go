package heavy

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
)

var (
	// writeOptions used by both non-Batch and Batch leveldb.Put
	writeOptions = &opt.WriteOptions{NoWriteMerge: optionNoWriteMerge, Sync: optionFsyncEnabled}

	// Key values prefixes to provide prefix-based sorting mechanism
	// Refer to README.md for overview idea

	// HeaderPrefix is the prefix to identify the Header
	HeaderPrefix = []byte{0x01}
	// TxPrefix is the prefix to identify Transactions
	TxPrefix = []byte{0x02}
	// HeightPrefix is the prefix to identify the Height
	HeightPrefix = []byte{0x03}
	// TxIDPrefix is the prefix to identify the Transaction ID
	TxIDPrefix = []byte{0x04}
	// KeyImagePrefix is the prefix to identify the Key Image
	KeyImagePrefix = []byte{0x05}
	// StatePrefix is the prefix to identify the State
	StatePrefix = []byte{0x06}
	// OutputKeyPrefix is the prefix to identify the Output
	OutputKeyPrefix = []byte{0x07}
	// BidValuesPrefix is the prefix to identify Bid Values
	BidValuesPrefix = []byte{0x08}
)

type transaction struct {
	writable bool
	db       *DB

	// Get/Has/Iterate calls must be applied into the snapshot only
	snapshot *leveldb.Snapshot

	// Put/Delete calls must be applied into a batch only. Transaction does
	// implement atomicity by a levelDB.Batch constructed during the
	// Transaction.
	batch  *leveldb.Batch
	closed bool
}

// StoreBlock stores the entire block data into storage. No validations are
// applied. Method simply stores the block data into LevelDB storage in an
// atomic way. That said, storage state changes only when Commit() is called on
// Transaction completion.
//
// See also the method body to get an idea of Key-Value data schemas.
//
// It is assumed that StoreBlock would be called much less times than Fetch*
// APIs. Based on that, extra indexing data is put to provide fast-read lookups
func (t transaction) StoreBlock(b *block.Block) error {

	if t.batch == nil {
		// t.batch is initialized only on a open, read-write transaction
		// (built with transaction.Update())
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
	t.put(key, value)

	if len(b.Txs) > math.MaxUint32 {
		return errors.New("too many transactions")
	}

	// Put block transaction data. A KV pair per a single transaction is added
	// into the store
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

		t.put(keys, entry)

		// Schema
		//
		// Key = TxIDPrefix + txID
		// Value = block.header.hash
		//
		// For the retrival of a single transaction by TxId

		t.put(append(TxIDPrefix, txID...), b.Header.Hash)

		// Schema
		//
		// Key = KeyImagePrefix + tx.input.KeyImage
		// Value = txID
		//
		// To make FetchKeyImageExists functioning
		for _, input := range tx.StandardTx().Inputs {
			t.put(append(KeyImagePrefix, input.KeyImage.Bytes()...), txID)
		}

		// Schema
		//
		// Key = OutputKeyPrefix + tx.output.PublicKey
		// Value = unlockheight
		//
		// To make FetchOutputKey functioning
		for i, output := range tx.StandardTx().Outputs {
			v := make([]byte, 8)
			// Only lock the first output, so that change outputs are
			// not affected.
			if i == 0 {
				binary.LittleEndian.PutUint64(v, tx.LockTime()+b.Header.Height)
			}
			t.put(append(OutputKeyPrefix, output.PubKey.P.Bytes()...), v)
		}

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
	value = b.Header.Hash
	t.put(key, value)

	// Key = StatePrefix
	// Value = Hash(chain tip)
	//
	// To support fetching  blockchain tip
	key = StatePrefix
	value = b.Header.Hash
	t.put(key, value)

	// Delete expired bid values
	key = BidValuesPrefix
	iterator := t.snapshot.NewIterator(util.BytesPrefix(key), nil)
	defer iterator.Release()

	for iterator.Next() {
		if len(iterator.Key()) != 9 {
			// Malformed key found, however we should not abort the entire
			// operation just because of it.
			log.WithFields(log.Fields{
				"process": "database",
				"key":     iterator.Key(),
			}).WithError(errors.New("bid values entry with malformed key found")).Errorln("error when iterating over bid values")
			// Let's remove it though, so that we don't keep logging errors
			// for the same entry.
			t.batch.Delete(iterator.Key())
			continue
		}

		height := binary.LittleEndian.Uint64(iterator.Key()[1:])
		if height < b.Header.Height {
			t.batch.Delete(iterator.Key())
		}
	}

	return iterator.Error()
}

// Commit writes a batch to LevelDB storage. See also fsyncEnabled variable
func (t *transaction) Commit() error {
	if !t.writable {
		return errors.New("read-only transaction cannot commit changes")
	}

	if t.closed {
		return errors.New("already closed transaction cannot commit changes")
	}

	return t.db.storage.Write(t.batch, writeOptions)
}

// Rollback is not used by database layer
func (t transaction) Rollback() error {
	t.batch.Reset()
	return nil
}

// Close releases the retrieved snapshot and resets any open batch(s). It must
// be called explicitly when a transaction is run in a unmanaged way
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

// FetchOutputExists checks if an output exists in the db
func (t transaction) FetchOutputExists(destkey []byte) (bool, error) {
	key := append(OutputKeyPrefix, destkey...)
	exists, err := t.snapshot.Has(key, nil)

	// goleveldb returns nilIfNotFound
	// see also nilIfNotFound in leveldb/db.go
	if !exists && err == nil {
		// overwrite error message
		err = database.ErrOutputNotFound
	}

	return exists, err
}

// FetchOutputUnlockHeight returns the unlockheight of an output
func (t transaction) FetchOutputUnlockHeight(destkey []byte) (uint64, error) {
	key := append(OutputKeyPrefix, destkey...)
	unlockHeightBytes, err := t.snapshot.Get(key, nil)
	if err != nil {
		return 0, err
	}

	if len(unlockHeightBytes) != 8 {
		return 0, errors.New("unlock height malformed")
	}

	// output unlock height is the first 8 bytes
	unlockHeight := binary.LittleEndian.Uint64(unlockHeightBytes[0:8])
	return unlockHeight, err
}

// FetchDecoys iterates over the outputs and fetches `numDecoys` amount
// of output public keys
func (t transaction) FetchDecoys(numDecoys int) []ristretto.Point {
	scanFilter := OutputKeyPrefix

	iterator := t.snapshot.NewIterator(util.BytesPrefix(scanFilter), nil)
	defer iterator.Release()

	decoysPubKeys := make([]ristretto.Point, 0, numDecoys)

	currentHeight, err := t.FetchCurrentHeight()
	if err != nil {
		log.Panic(err)
	}

	for iterator.Next() {
		// We only take unlocked decoys
		unlockHeight := binary.LittleEndian.Uint64(iterator.Value())
		if unlockHeight > currentHeight {
			continue
		}

		// Output public key is the iterator key minus the `OutputKeyPrefix`
		// (1 byte)
		value := iterator.Key()[1:]

		var p ristretto.Point
		var pBytes [32]byte
		copy(pBytes[:], value)
		p.SetBytes(&pBytes)

		decoysPubKeys = append(decoysPubKeys, p)
		if len(decoysPubKeys) == numDecoys {
			break
		}
	}

	return decoysPubKeys
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

func (t transaction) FetchBlockTxs(hashHeader []byte) ([]transactions.Transaction, error) {

	scanFilter := append(TxPrefix, hashHeader...)
	tempTxs := make(map[uint32]transactions.Transaction)

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
	resultTxs := make([]transactions.Transaction, len(tempTxs))
	for k, v := range tempTxs {
		resultTxs[k] = v
	}

	// Let's ensure coinbase tx is here
	if len(resultTxs) > 0 {
		if resultTxs[0].Type() != transactions.CoinbaseType {
			return resultTxs, errors.New("missing coinbase tx")
		}
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

func (t transaction) FetchBlockTxByHash(txID []byte) (transactions.Transaction, uint32, []byte, error) {

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
		n, err := reader.ReadAt(fetchedTxID[:], int64(len(scanFilter)))

		// We should always be capable of reading the TxID from the KEY
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

// FetchKeyImageExists checks if the KeyImage exists. If so, it also returns the
// hash of its corresponding tx.
//
// Due to performance concerns, the found tx is not verified. By explicitly
// calling FetchBlockTxByHash, a consumer can check if the tx is real
func (t transaction) FetchKeyImageExists(keyImage []byte) (bool, []byte, error) {

	key := append(KeyImagePrefix, keyImage...)
	txID, err := t.snapshot.Get(key, nil)

	if err != nil {
		if err == leveldb.ErrNotFound {
			// overwrite error message
			err = database.ErrKeyImageNotFound
		}
		return false, nil, err
	}

	return true, txID, nil
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

func (t transaction) FetchState() (*database.State, error) {
	key := StatePrefix
	value, err := t.snapshot.Get(key, nil)
	if err == leveldb.ErrNotFound || len(value) == 0 {
		// overwrite error message
		err = database.ErrStateNotFound
	}

	if err != nil {
		return nil, err
	}

	return &database.State{TipHash: value}, nil
}

func (t transaction) FetchCurrentHeight() (uint64, error) {
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

func (t transaction) StoreBidValues(d, k []byte, lockTime uint64) error {
	// First, delete the old values (if any)
	heightBytes := make([]byte, 8)
	currentHeight, err := t.FetchCurrentHeight()
	if err != nil {
		return err
	}

	// NOTE: this expiry height is not accurate, and is just an
	// approximation. On average, it will vary only a few blocks, but
	// we can not know beforehand when a bid transaction is accepted.
	binary.LittleEndian.PutUint64(heightBytes, lockTime+currentHeight)
	key := append(BidValuesPrefix, heightBytes...)
	t.put(key, append(d, k...))
	return nil
}

func (t transaction) FetchBidValues() ([]byte, []byte, error) {
	key := BidValuesPrefix
	iterator := t.snapshot.NewIterator(util.BytesPrefix(key), nil)
	defer iterator.Release()

	// Let's always return the bid values with the lowest height as
	// those are most likely to be valid.
	lowestSeen := uint64(1<<64 - 1)
	var value []byte
	for iterator.Next() {
		if len(iterator.Key()) != 9 {
			// Malformed key found, however we should not abort the entire
			// operation just because of it.
			log.WithFields(log.Fields{
				"process": "database",
				"key":     iterator.Key(),
			}).WithError(errors.New("bid values entry with malformed key found")).Errorln("error when iterating over bid values")
			continue
		}

		// height is the last 8 bytes
		height := binary.LittleEndian.Uint64(iterator.Key()[1:])

		if height < lowestSeen {
			lowestSeen = height
			value = iterator.Value()
		}
	}

	if err := iterator.Error(); err != nil {
		return nil, nil, err
	}

	// Let's avoid any runtime panics by doing a sanity check on the value length before
	if len(value) != 64 {
		return nil, nil, errors.New("bid values non-existent or incorrectly encoded")
	}

	return value[0:32], value[32:64], nil
}

// FetchBlockHeightSince uses binary search to find a block height
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

// ClearDatabase will wipe all of the data currently in the database.
func (t transaction) ClearDatabase() error {
	iter := t.snapshot.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		t.batch.Delete(iter.Key())
	}

	return iter.Error()
}
