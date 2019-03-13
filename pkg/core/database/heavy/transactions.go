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

var (
	// Explicitly show we don't want fsync on each batch write
	fsyncEnabled = false

	// TODO:
	optionNoWriteMerge = false
	writeOptions       = &opt.WriteOptions{NoWriteMerge: optionNoWriteMerge, Sync: fsyncEnabled}

	// ByteOrder to be used on any internal en/decoding
	byteOrder = binary.LittleEndian

	// Prefixes
	HEADER_PREFIX = []byte{0x11}
	TX_PREFIX     = []byte{0x22}
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
	batch  *leveldb.Batch
	closed bool
}

// GetBlockHeaderByHash gives the block header from the hash
func (t Tx) GetBlockHeaderByHash(hash []byte) (*block.Header, error) {

	// only a dummy get
	h := &block.Header{}
	value, err := t.snapshot.Get(hash, nil)

	if err != nil {
		return nil, err
	}

	h.Height = binary.LittleEndian.Uint64(value)
	return h, err
}

// TODO:  tests
// Store the entire block data into storage. No validations are applied.
// Method simply stores the block data into Tx Batch
//
//	Block.Header.Hash -> Encoded(Block.Header.Fields)
//  TX + block.Header.Hash +
func (t Tx) StoreBlock(block *block.Block) error {

	// TODO: Do we need to Check HasBlock( blockHeaderFields )

	// Put KV (Key-Value) pair
	//
	// KV Schema:
	//
	// KEY = HEADER_PREFIX + Block.Header.Hash
	// VALUE = Encoded( Block.Fields )

	// Block.Header.Hash -> Block.Header.Fields()
	blockHeaderFields := new(bytes.Buffer)
	if err := block.Header.EncodeHashable(blockHeaderFields); err != nil {
		return err
	}

	key := append(HEADER_PREFIX, block.Header.Hash...)
	value := blockHeaderFields.Bytes()
	t.batch.Put(key, value)

	// Put block transaction data.
	for index, v := range block.Txs {

		tx := v.(*transactions.Stealth)

		// Put KV pair
		// KV Schema:
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

		t.batch.Put(key, value)
	}

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
func (t Tx) decodeBlockTx(buffer []byte) (*transactions.Stealth, uint32, error) {
	// TODO:
	return nil, 0, nil
}

// writeUint32 Tx utility to use a common byteOrder on internal encoding
func (t Tx) writeUint32(w io.Writer, value uint32) error {
	var b [4]byte
	byteOrder.PutUint32(b[:], value)
	_, err := w.Write(b[:])
	return err
}

/*
// WriteHeader writes dummy data
func (t Tx) WriteHeader(header *block.Header) error {

	//  only a dummy put
	key := header.Hash
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value[0:], header.Height)
	t.batch.Put(key, value)

	return nil
}
*/

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
	return nil
}

// Close Releases the retrieved snapshot. Do not forget it when
// unmanaged Tx is used
func (t *Tx) Close() {
	t.snapshot.Release()
	t.closed = true
}

func (t Tx) FetchBlockExists(header *block.Header) (bool, error) {
	// TODO: Do we need special readOptions here
	key := append(HEADER_PREFIX, header.Hash...)
	result, err := t.snapshot.Has(key, nil)
	return result, err
}

func (t Tx) FetchBlockHeader(hash []byte) (*block.Header, error) {
	//TODO:
	return nil, nil
}

func (t Tx) FetchBlockTransactions(hashHeader []byte) ([]merkletree.Payload, error) {

	scanFilter := append(TX_PREFIX, hashHeader...)
	tempTxs := make(map[uint32]merkletree.Payload)

	// Read all the transactions, first do a prefix scan on TX + header hash
	iterator := t.snapshot.NewIterator(util.BytesPrefix(scanFilter), nil)
	defer iterator.Release()

	var txCount uint32
	for iterator.Next() {
		key := iterator.Value()
		value, err := t.snapshot.Get(key, nil)
		if err != nil {
			return nil, err
		}

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
