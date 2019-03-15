package heavy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"io"
)

const (
	// Atomic updates enabled/disabled
	atomicUpdateEnabled = true

	// Explicitly show we don't want fsync on each batch write
	fsyncEnabled = false

	// TODO: NoWriteMerge is needed?
	optionNoWriteMerge = false
)

var (
	// writeOptions used by both non-Batch and Batch leveldb.Put
	writeOptions = &opt.WriteOptions{NoWriteMerge: optionNoWriteMerge, Sync: fsyncEnabled}

	// ByteOrder to be used on any internal en/decoding
	byteOrder = binary.LittleEndian

	// Key values prefixes
	HEADER_PREFIX = []byte{0x11}
	TX_PREFIX     = []byte{0x22}
	HEIGHT_PREFIX = []byte{0x33}
)

// A writable transaction would Put/Delete into leveldb.Batch only
// to achieve atomicity on changing blockchain state
type Tx struct {
	writable bool
	db       *DB

	// Get/Has/Iterate calls must be applied into the snapshot only
	snapshot *leveldb.Snapshot

	// Put/Delete calls must be applied into the batch only
	// Tx does implement atomicity by batch levelDB.
	// Batch is constructed during the Tx but committed only on Tx completion
	// See also (t *Tx) Commit()
	// TODO: Safe in concurrent execution?
	//		Can we have safely multiple batches open at the same time
	//
	batch  *leveldb.Batch
	closed bool
}

// TODO:  tests
// Store the entire block data into storage. No validations are applied.
// Method simply stores the block data into Tx Batch via atomic update
//
// Storage state change is held only when Commit() is called on Tx completion
// Block.Header.Hash -> Encoded(Block.Header.Fields)
// TX + block.Header.Hash +
func (t Tx) StoreBlock(block *block.Block) error {

	if atomicUpdateEnabled {
		if t.batch == nil {
			return errors.New("StoreBlock on read-only tx")
		}
	}

	// TODO: Do we need to Check HasBlock( blockHeaderFields )

	// Key-Value Schema:
	//
	// KEY = HEADER_PREFIX + Block.Header.Hash
	// VALUE = Encoded( Block.Fields )

	// Block.Header.Hash -> Block.Header.Fields()
	blockHeaderFields := new(bytes.Buffer)
	if err := block.Header.Encode(blockHeaderFields); err != nil {
		return err
	}

	key := append(HEADER_PREFIX, block.Header.Hash...)
	value := blockHeaderFields.Bytes()
	t.Put(key, value)

	// Put block transaction data.
	// A KV pair per a single transaction is added into the store
	for index, v := range block.Txs {

		tx := v.(*transactions.Stealth)

		// Key-Value Schema:
		//
		// KEY = TX_PREFIX + Block.Header.Hash + Tx.R
		// VALUE = Encoded(index) + Encoded( Block.Transaction[index] )
		//
		// For the retrival of transactions data by Header.Hash

		key := append(TX_PREFIX, block.Header.Hash...)
		key = append(key, tx.R...)
		value, err := t.encodeBlockTx(tx, uint32(index))

		if err != nil {
			return err
		}

		t.Put(key, value)
	}

	// Key-Value Schema:
	// KEY = HEIGHT_PREFIX + Block.Header.Height
	// VALUE = Block.Header.Hash
	//
	// To support fast header lookup by height

	heightBuf := new(bytes.Buffer)

	// Append index value
	if err := t.writeUint64(heightBuf, block.Header.Height); err != nil {
		return err
	}

	key = append(HEIGHT_PREFIX, heightBuf.Bytes()...)
	value = block.Header.Hash
	t.Put(key, value)

	return nil
}

// encodeBlockTx Returns Tx Bytes prefixed with Tx Index value
func (t Tx) encodeBlockTx(tx *transactions.Stealth, index uint32) ([]byte, error) {

	buf := new(bytes.Buffer)

	// Append index value
	if err := t.writeUint32(buf, index); err != nil {
		return nil, err
	}

	err := tx.Encode(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// encodeBlockTx Returns Tx Bytes prefixed with Tx Index value
func (t Tx) decodeBlockTx(data []byte) (*transactions.Stealth, uint32, error) {

	buf := bytes.NewReader(data)

	// Append index value
	var index uint32
	if err := t.readUint32(buf, &index); err != nil {
		return nil, 0, err
	}

	tx := &transactions.Stealth{}
	if err := tx.Decode(buf); err != nil {
		return nil, 0, err
	}

	return tx, index, nil
}

// writeUint32 Tx utility to use a common byteOrder on internal encoding
func (t Tx) writeUint32(w io.Writer, value uint32) error {
	var b [4]byte
	byteOrder.PutUint32(b[:], value)
	_, err := w.Write(b[:])
	return err
}

// ReadUint32 will read four bytes and convert them to a uint32
// from the specified byte order. The result is put into v.
func (t Tx) readUint32(r io.Reader, v *uint32) error {
	var b [4]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return err
	}
	*v = byteOrder.Uint32(b[:])
	return nil
}

// writeUint32 Tx utility to use a common byteOrder on internal encoding
func (t Tx) writeUint64(w io.Writer, value uint64) error {
	var b [8]byte
	byteOrder.PutUint64(b[:], value)
	_, err := w.Write(b[:])
	return err
}

// Commit writes a batch to LevelDB storage.
// See also fsyncEnabled variable
func (t *Tx) Commit() error {
	if !t.writable {
		return errors.New("read-only transaction cannot commit changes")
	}

	if t.closed {
		return errors.New("already closed transaction cannot commit changes")
	}

	return t.db.storage.Write(t.batch, writeOptions)
}

func (t Tx) Rollback() error {
	// Achieved already by deprecating the leveldb.Batch
	t.batch.Reset()
	return nil
}

// Close Releases the retrieved snapshot. Do not forget it when
// unmanaged Tx is used
func (t *Tx) Close() {
	t.snapshot.Release()
	t.closed = true
}

func (t Tx) FetchBlockExists(hash []byte) (bool, error) {
	// TODO: Do we need special readOptions here
	key := append(HEADER_PREFIX, hash...)
	result, err := t.snapshot.Has(key, nil)
	return result, err
}

func (t Tx) FetchBlockHeader(hash []byte) (*block.Header, error) {

	key := append(HEADER_PREFIX, hash...)
	value, err := t.snapshot.Get(key, nil)

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

func (t Tx) FetchBlockTransactions(hashHeader []byte) ([]merkletree.Payload, error) {

	scanFilter := append(TX_PREFIX, hashHeader...)
	tempTxs := make(map[uint32]merkletree.Payload)

	// Read all the transactions, first do a prefix scan on TX + header hash
	iterator := t.snapshot.NewIterator(util.BytesPrefix(scanFilter), nil)
	defer iterator.Release()

	var txCount uint32
	for iterator.Next() {
		value := iterator.Value()
		tx, index, err := t.decodeBlockTx(value)

		if err != nil {
			return nil, err
		}

		tempTxs[index] = tx
		txCount++
	}

	// Reorder Tx slice as per retrieved indeces
	resultTxs := make([]merkletree.Payload, txCount)
	for k, v := range tempTxs {
		resultTxs[k] = v
	}

	return resultTxs, nil
}

func (t Tx) FetchBlockHashByHeight(height uint64) ([]byte, error) {

	// TODO: duplicated code
	heightBuf := new(bytes.Buffer)

	// Append index value
	if err := t.writeUint64(heightBuf, height); err != nil {
		return nil, err
	}

	key := append(HEIGHT_PREFIX, heightBuf.Bytes()...)
	value, err := t.snapshot.Get(key, nil)

	if err != nil {
		return nil, err
	}

	return value, nil
}

func (t Tx) Put(key []byte, value []byte) {

	if atomicUpdateEnabled {
		if t.batch != nil {
			t.batch.Put(key, value)
		}
	} else {
		_ = t.db.storage.Put(key, value, writeOptions)
	}
}
