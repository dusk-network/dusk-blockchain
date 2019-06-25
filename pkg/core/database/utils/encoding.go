package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

var (
	byteOrder = binary.LittleEndian
)

// encodeBlockTx tries to serialize type, index and encoded value of transactions.Transaction
func EncodeBlockTx(tx transactions.Transaction, txIndex uint32) ([]byte, error) {

	buf := new(bytes.Buffer)

	// Write tx type as first field
	if err := buf.WriteByte(byte(tx.Type())); err != nil {
		return nil, err
	}

	// Write index value as second field.

	// golevedb is ordering keys lexicographically. That said, the order of the
	// stored KV is not the order of inserting
	if err := WriteUint32(buf, txIndex); err != nil {
		return nil, err
	}

	// Write transactions.Transaction bytes
	err := tx.Encode(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DecodeBlockTx(data []byte, typeFilter transactions.TxType) (transactions.Transaction, uint32, error) {

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
	if err := ReadUint32(reader, &txIndex); err != nil {
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
func WriteUint32(w io.Writer, value uint32) error {
	var b [4]byte
	byteOrder.PutUint32(b[:], value)
	_, err := w.Write(b[:])
	return err
}

// ReadUint32 will read four bytes and convert them to a uint32 from the Tx
// byteOrder. The result is put into v.
func ReadUint32(r io.Reader, v *uint32) error {
	var b [4]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return err
	}
	*v = byteOrder.Uint32(b[:])
	return nil
}

// writeUint64 Tx utility to use a common byteOrder on internal encoding
func WriteUint64(w io.Writer, value uint64) error {
	var b [8]byte
	byteOrder.PutUint64(b[:], value)
	_, err := w.Write(b[:])
	return err
}
