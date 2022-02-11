// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
)

var byteOrder = binary.LittleEndian

// EncodeBlockTx tries to serialize type, index and encoded value of transactions.ContractCall.
func EncodeBlockTx(tx transactions.ContractCall, txIndex uint32) ([]byte, error) {
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

	// Write tx gas spent as third field
	if err := WriteUint64(buf, tx.GasSpent()); err != nil {
		return nil, err
	}

	// Write transactions.ContractCall bytes
	err := transactions.Marshal(buf, tx)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeBlockTx tries to deserialize the type, index and decoded value of a tx.
func DecodeBlockTx(data []byte, typeFilter transactions.TxType) (transactions.ContractCall, uint32, error) {
	txIndex := uint32(math.MaxUint32)

	tx := transactions.NewTransaction()
	reader := bytes.NewBuffer(data)

	// Peek the type from the first byte
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
	if e := ReadUint32(reader, &txIndex); e != nil {
		return nil, txIndex, e
	}

	// Read gasSpent field
	var gasSpent uint64
	if e := ReadUint64(reader, &gasSpent); e != nil {
		return nil, txIndex, e
	}

	if e := transactions.Unmarshal(reader, tx); e != nil {
		return tx, txIndex, err
	}

	cc, err := transactions.UpdateGasSpent(tx, gasSpent)
	if err != nil {
		return tx, txIndex, err
	}

	return cc, txIndex, err
}

// WriteUint32 Tx utility to use a Tx byteOrder on internal encoding.
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

// WriteUint64 Tx utility to use a common byteOrder on internal encoding.
func WriteUint64(w io.Writer, value uint64) error {
	var b [8]byte

	byteOrder.PutUint64(b[:], value)

	_, err := w.Write(b[:])
	return err
}

// ReadUint64 will read four bytes and convert them to a uint64 from the Tx
// byteOrder. The result is put into v.
func ReadUint64(r io.Reader, v *uint64) error {
	var b [8]byte

	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return err
	}

	*v = byteOrder.Uint64(b[:])
	return nil
}
