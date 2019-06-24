package lite

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

var (
	byteOrder = binary.LittleEndian
)

type transaction struct {
	writable bool
	db       *DB
	batch    memdb
}

func (t *transaction) StoreBlock(b *block.Block) error {

	// Map header.Hash to block.Block
	buf := new(bytes.Buffer)
	if err := b.Encode(buf); err != nil {
		return err
	}
	t.batch[blocksInd][toKey(b.Header.Hash)] = buf.Bytes()

	// Map Height to block.Block
	// TODO: use slice
	// t.batch[heightInd][toKey(b.Header.Height)] = buf.Bytes()

	// Map txId to transactions.Transaction
	for i, tx := range b.Txs {

		txID, err := tx.CalculateHash()
		if err != nil {
			return err
		}

		if len(txID) == 0 {
			return fmt.Errorf("empty chain tx id")
		}

		data, err := t.encodeBlockTx(tx, uint32(i))
		if err != nil {
			return err
		}

		t.batch[txsInd][toKey(txID)] = data
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

	b := block.Block{}
	if err := b.Decode(bytes.NewReader(data)); err != nil {
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

	b := block.Block{}
	if err := b.Decode(bytes.NewReader(data)); err != nil {
		return nil, err
	}

	return b.Txs, nil
}

func (t transaction) FetchBlockHashByHeight(height uint64) ([]byte, error) {
	return nil, database.ErrBlockNotFound
}

func (t transaction) FetchBlockTxByHash(txID []byte) (transactions.Transaction, uint32, []byte, error) {

	var data []byte
	var exists bool
	if data, exists = t.db.storage[txsInd][toKey(txID)]; !exists {
		return nil, 0, nil, database.ErrTxNotFound
	}

	tx, txIndex, err := t.decodeBlockTx(data, database.AnyTxType)
	if err != nil {
		return nil, 0, nil, database.ErrTxNotFound
	}

	// TODO: hashHeader the tx belongs to
	return tx, txIndex, nil, err
}

// FetchKeyImageExists checks if the KeyImage exists. If so, it also returns the
// hash of its corresponding tx.
//
// Due to performance concerns, the found tx is not verified. By explicitly
// calling FetchBlockTxByHash, a consumer can check if the tx is real
func (t transaction) FetchKeyImageExists(keyImage []byte) (bool, []byte, error) {

	// TODO: Map keyImage to txID
	var txID []byte
	var exists bool
	if txID, exists = t.db.storage[keyImagesInd][toKey(keyImage)]; !exists {
		return false, nil, database.ErrKeyImageNotFound
	}

	return true, txID, nil
}

// StoreCandidateBlock stores a candidate block to be proposed in next consensus
// round. it overwrites an entry of block with same height
func (t *transaction) StoreCandidateBlock(b *block.Block) error {
	buf := new(bytes.Buffer)
	if err := b.Encode(buf); err != nil {
		return err
	}
	t.batch[candidatesTableIndex][toKey(b.Header.Hash)] = buf.Bytes()

	return nil
}

// FetchCandidateBlock fetches a candidate block by hash
func (t transaction) FetchCandidateBlock(hash []byte) (*block.Block, error) {

	var data []byte
	var exists bool
	if data, exists = t.db.storage[candidatesTableIndex][toKey(hash)]; !exists {
		return nil, database.ErrBlockNotFound
	}

	b := &block.Block{}
	if err := b.Decode(bytes.NewReader(data)); err != nil {
		return nil, err
	}

	return b, nil
}

func (t transaction) DeleteCandidateBlocks(maxHeight uint64) (uint32, error) {

	// TODO: delete(t.db.storage[candidatesTableIndex])
	return 0, nil
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

// TODO: duplicated code

// encodeBlockTx tries to serialize type, index and encoded value of transactions.Transaction
func (t transaction) encodeBlockTx(tx transactions.Transaction, txIndex uint32) ([]byte, error) {

	buf := new(bytes.Buffer)

	// Write tx type as first field
	if err := buf.WriteByte(byte(tx.Type())); err != nil {
		return nil, err
	}

	// Write index value as second field.

	// golevedb is ordering keys lexicographically. That said, the order of the
	// stored KV is not the order of inserting
	if err := t.writeUint32(buf, txIndex); err != nil {
		return nil, err
	}

	// Write transactions.Transaction bytes
	err := tx.Encode(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t transaction) decodeBlockTx(data []byte, typeFilter transactions.TxType) (transactions.Transaction, uint32, error) {

	txIndex := uint32(math.MaxUint32)

	var tx transactions.Transaction
	reader := bytes.NewReader(data)

	// Peak the type from the first byte
	typeBytes, err := reader.ReadByte()
	if err != nil {
		return nil, txIndex, err
	}
	txReadType := transactions.TxType(typeBytes)

	if typeFilter != database.AnyTxType {
		// Do not read and decode the rest of the bytes if the transaction type
		// is not same as typeFilter
		if typeFilter != txReadType {
			return nil, txIndex, fmt.Errorf("tx of type %d not found", typeFilter)
		}
	}

	// Read tx index field
	if err := t.readUint32(reader, &txIndex); err != nil {
		return nil, txIndex, err
	}

	switch txReadType {
	case transactions.StandardType:
		tx = &transactions.Standard{}
		err := tx.Decode(reader)
		if err != nil {
			return nil, txIndex, err
		}
	case transactions.TimelockType:
		tx = &transactions.TimeLock{}
		err := tx.Decode(reader)
		if err != nil {
			return nil, txIndex, err
		}
	case transactions.BidType:
		tx = &transactions.Bid{}
		err := tx.Decode(reader)
		if err != nil {
			return nil, txIndex, err
		}
	case transactions.StakeType:
		tx = &transactions.Stake{}
		err := tx.Decode(reader)
		if err != nil {
			return nil, txIndex, err
		}
	case transactions.CoinbaseType:
		tx = &transactions.Coinbase{}
		err := tx.Decode(reader)
		if err != nil {
			return nil, txIndex, err
		}
	default:
		return nil, txIndex, fmt.Errorf("unknown transaction type: %d", txReadType)
	}

	return tx, txIndex, nil
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
