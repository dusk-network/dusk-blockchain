// Serialization functions for integers

// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"bytes"
	"encoding/binary"
)

// ReadUint8 will read a single byte into v.
func ReadUint8(r *bytes.Buffer, v *uint8) error {
	var b [1]byte
	if _, err := r.Read(b[:]); err != nil {
		return err
	}
	*v = b[0]
	return nil
}

// ReadUint16LE will read two bytes and convert them to a uint16
// assuming little-endian byte order. The result is put into v.
func ReadUint16LE(r *bytes.Buffer, v *uint16) error {
	var b [2]byte
	if _, err := r.Read(b[:]); err != nil {
		return err
	}
	*v = binary.LittleEndian.Uint16(b[:])
	return nil
}

// ReadUint32LE will read four bytes and convert them to a uint32
// assuming little-endian byte order. The result is put into v.
func ReadUint32LE(r *bytes.Buffer, v *uint32) error {
	var b [4]byte
	if _, err := r.Read(b[:]); err != nil {
		return err
	}
	*v = binary.LittleEndian.Uint32(b[:])
	return nil
}

// ReadUint64LE will read eight bytes and convert them to a uint64
// assuming little-endian byte order. The result is put into v.
func ReadUint64LE(r *bytes.Buffer, v *uint64) error {
	var b [8]byte
	if _, err := r.Read(b[:]); err != nil {
		return err
	}
	*v = binary.LittleEndian.Uint64(b[:])
	return nil
}

// WriteUint8 will write a single byte.
func WriteUint8(w *bytes.Buffer, v uint8) error {
	_, err := w.Write([]byte{v})
	return err
}

// WriteUint16LE will write two bytes in little-endian byte order.
func WriteUint16LE(w *bytes.Buffer, v uint16) error {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// WriteUint32LE will write four bytes in little-endian byte order.
func WriteUint32LE(w *bytes.Buffer, v uint32) error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// WriteUint64LE will write eight bytes in little-endian byte order.
func WriteUint64LE(w *bytes.Buffer, v uint64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	_, err := w.Write(b[:])
	return err
}
