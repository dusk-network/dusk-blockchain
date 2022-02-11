// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

// Variable length data serialization functions

package encoding

import (
	"bytes"
	"fmt"
)

// ReadVarBytes will read a CompactSize int denoting the length, then
// proceeds to read that amount of bytes from r into b.
func ReadVarBytes(r *bytes.Buffer, b *[]byte) error {
	c, err := ReadVarInt(r)
	if err != nil {
		return err
	}

	// We reject reading any data that has a length greater than
	// math.MaxInt32, to avoid out of memory errors.
	if c > uint64(r.Len()) {
		return fmt.Errorf("attempting to decode data which is too large %d", c)
	}

	*b = make([]byte, c)
	if _, err := r.Read(*b); err != nil {
		return err
	}
	return nil
}

// ReadVarBytesUint32LE will read a CompactSize int denoting the length, then
// proceeds to read that amount of bytes from r into b.
func ReadVarBytesUint32LE(r *bytes.Buffer, b *[]byte) error {
	var c uint32
	if err := ReadUint32LE(r, &c); err != nil {
		return err
	}

	// We reject reading any data that has a length greater than
	// math.MaxInt32, to avoid out of memory errors.
	if c > uint32(r.Len()) {
		return fmt.Errorf("attempting to decode data which is too large %d", c)
	}

	*b = make([]byte, c)
	if _, err := r.Read(*b); err != nil {
		return err
	}
	return nil
}

// WriteVarBytes will serialize a CompactSize int denoting the length, then
// proceeds to write b into w.
func WriteVarBytes(w *bytes.Buffer, b []byte) error {
	if err := WriteVarInt(w, uint64(len(b))); err != nil {
		return err
	}

	_, err := w.Write(b)
	return err
}

// WriteVarBytesUint32 will serialize a CompactSize int denoting the length, then
// proceeds to write b into w.
func WriteVarBytesUint32(w *bytes.Buffer, b []byte) error {
	if err := WriteUint32LE(w, uint32(len(b))); err != nil {
		return err
	}

	_, err := w.Write(b)
	return err
}

// Convenience functions for strings. They will point to the functions above and
// handle type conversion.

// ReadString reads the data with ReadVarBytes and returns it as a string
// by simple type conversion.
func ReadString(r *bytes.Buffer) (string, error) {
	var b []byte
	if err := ReadVarBytes(r, &b); err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteString will write string s as a slice of bytes through WriteVarBytes.
func WriteString(w *bytes.Buffer, s string) error {
	return WriteVarBytes(w, []byte(s))
}
