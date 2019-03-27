package heavy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"io"
	"math"
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

	// ByteOrder to be used on any internal en/decoding
	byteOrder = binary.LittleEndian

	// Key values prefixes to provide prefix-based sorting mechanism
	HeaderPrefix   = []byte{0x01}
	TxPrefix       = []byte{0x02}
	HeightPrefix   = []byte{0x03}
	TxIdPrefix     = []byte{0x04}
	KeyImagePrefix = []byte{0x05}
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
func (t transaction) StoreBlock(block *block.Block) error {

	if t.batch == nil {
		// t.batch is initialized only on a open, read-write transaction
		// (built with transaction.Update())
		return errors.New("StoreBlock cannot be called on read-only transaction")
	}

	// Schema Key = HeaderPrefix + block.header.hash Value =
	// encoded(block.fields)

	blockHeaderFields := new(bytes.Buffer)
	if err := block.Header.Encode(blockHeaderFields); err != nil {
		return err
	}

	key := append(HeaderPrefix, block.Header.Hash...)
	value := blockHeaderFields.Bytes()
	t.put(key, value)

	if len(block.Txs) > math.MaxUint32 {
		return errors.New("too many transactions")
	}

	// Put block transaction data. A KV pair per a single transaction is added
	// into the store
	for index, v := range block.Txs {

		tx := v.(*transactions.Stealth)

		// Schema
		//
		// Key = TxPrefix + block.header.hash + Tx.R
		// Value = index + block.transaction[index]
		//
		// For the retrival of transactions data by block.header.hash

		key := append(TxPrefix, block.Header.Hash...)
		key = append(key, tx.R...)
		value, err := t.encodeBlockTx(tx, uint32(index))

		if err != nil {
			return err
		}

		t.put(key, value)

		// Schema
		//
		// Key = TxPrefix + Tx.R
		// Value = block.header.hash
		//
		// For the retrival of a single transaction by TxId
		t.put(append(TxIdPrefix, tx.R...), block.Header.Hash)

		// Schema
		//
		// Key = KeyImagePrefix + tx.input.KeyImage
		// Value = tx.R
		//
		// To make FetchKeyImageExists functioning
		inputs := t.getInputs(tx)
		for _, input := range inputs {
			t.put(append(KeyImagePrefix, input.KeyImage...), tx.R)
		}

	}

	// Key = HeightPrefix + block.header.height
	// Value = block.header.hash
	//
	// To support fast header lookup by height

	heightBuf := new(bytes.Buffer)

	// Append height value
	if err := t.writeUint64(heightBuf, block.Header.Height); err != nil {
		return err
	}

	key = append(HeightPrefix, heightBuf.Bytes()...)
	value = block.Header.Hash
	t.put(key, value)

	return nil
}

// encodeBlockTx tries to serialize index and bytes of *transactions.Stealth
func (t transaction) encodeBlockTx(tx *transactions.Stealth, index uint32) ([]byte, error) {

	buf := new(bytes.Buffer)

	// Write index value
	if err := t.writeUint32(buf, index); err != nil {
		return nil, err
	}

	// Write transactions.Stealth bytes
	err := tx.Encode(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decodeBlockTx tries to deserialize index and bytes of a transaction.Stealth
func (t transaction) decodeBlockTx(data []byte) (*transactions.Stealth, uint32, error) {

	buf := bytes.NewReader(data)

	// Read index value
	var index uint32
	if err := t.readUint32(buf, &index); err != nil {
		return nil, 0, err
	}

	// Read transaction bytes
	tx := &transactions.Stealth{}
	if err := tx.Decode(buf); err != nil {
		return nil, 0, err
	}

	return tx, index, nil
}

// writeUint32 Tx utility to use a Tx byteOrder on internal encoding
func (t transaction) writeUint32(w io.Writer, value uint32) error {
	var b [4]byte
	byteOrder.PutUint32(b[:], value)
	_, err := w.Write(b[:])
	return err
}

// ReadUint32 will read four bytes and convert them to a uint32 from the Tx
// byteOrder. The result is put into v.
func (t transaction) readUint32(r io.Reader, v *uint32) error {
	var b [4]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return err
	}
	*v = byteOrder.Uint32(b[:])
	return nil
}

// writeUint32 Tx utility to use a common byteOrder on internal encoding
func (t transaction) writeUint64(w io.Writer, value uint64) error {
	var b [8]byte
	byteOrder.PutUint64(b[:], value)
	_, err := w.Write(b[:])
	return err
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

	header := new(block.Header)
	err = header.Decode(bytes.NewReader(value))

	if err != nil {
		return nil, err
	}

	return header, nil
}

func (t transaction) FetchBlockTxs(hashHeader []byte) ([]merkletree.Payload, error) {

	scanFilter := append(TxPrefix, hashHeader...)
	tempTxs := make(map[uint32]merkletree.Payload)

	// Read all the transactions that belong to a single block
	// Scan filter = TX_PREFIX + block.header.hash
	iterator := t.snapshot.NewIterator(util.BytesPrefix(scanFilter), nil)
	defer iterator.Release()

	for iterator.Next() {
		value := iterator.Value()
		tx, index, err := t.decodeBlockTx(value)

		if err != nil {
			return nil, err
		}

		tempTxs[index] = tx
	}

	// Reorder Tx slice as per retrieved indeces
	resultTxs := make([]merkletree.Payload, len(tempTxs))
	for k, v := range tempTxs {
		resultTxs[k] = v
	}

	return resultTxs, nil
}

func (t transaction) FetchBlockHashByHeight(height uint64) ([]byte, error) {

	// Get height bytes
	heightBuf := new(bytes.Buffer)
	if err := t.writeUint64(heightBuf, height); err != nil {
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
		panic("leveldb batch is unreachable")
	}
}

func (t transaction) FetchBlockTxByHash(txID []byte) (merkletree.Payload, uint32, []byte, error) {

	var txIndex uint32 = math.MaxUint32

	// Fetch the block header hash that this Tx belongs to
	key := append(TxIdPrefix, txID...)
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
		value := iterator.Value()
		tx, index, err := t.decodeBlockTx(value)

		if err == nil {
			if bytes.Equal(tx.R, txID) {
				return tx, index, hashHeader, nil
			}
		}
	}

	// If we get here, the transaction is available but something went wrong
	// with decoding the transaction

	return nil, txIndex, nil, errors.New("block tx is available but fetching it fails")
}

func (t transaction) FetchKeyImageExists(keyImage []byte) (exists bool, txID []byte, err error) {
	err = database.ErrKeyImageNotFound

	key := append(KeyImagePrefix, keyImage...)
	iterator := t.snapshot.NewIterator(util.BytesPrefix(key), nil)
	defer iterator.Release()

	for iterator.Next() {
		// Get txID that keyImage is assigned to
		txID = iterator.Value()
		// Ensure that the keyImage belongs to an input of a real tx
		tx, _, hashHeader, err := t.FetchBlockTxByHash(txID)
		if tx != nil && err == nil && hashHeader != nil {
			exists = true
			return exists, txID, err
		}
	}

	return exists, txID, err
}

// getInputs retrieves a slice of the inputs of Standard and Timelock txs
func (t transaction) getInputs(tx *transactions.Stealth) []*transactions.Input {

	switch tx.Type {
	case transactions.StandardType:
		typeInfo := tx.TypeInfo.(*transactions.Standard)
		return typeInfo.Inputs

	case transactions.TimelockType:
		typeInfo := tx.TypeInfo.(*transactions.Timelock)
		return typeInfo.Inputs
	}
	return nil
}
