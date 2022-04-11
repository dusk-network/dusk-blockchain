// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

// Serialization functions for a variety of data structures, such as hashes,
// signatures, and booleans.

package encoding

import (
	"bytes"
	"errors"
	"fmt"
)

// ReadBool will read a single byte from r, turn it into a bool
// and return it.
func ReadBool(r *bytes.Buffer, b *bool) error {
	var v uint8

	if err := ReadUint8(r, &v); err != nil {
		return err
	}

	*b = v != 0
	return nil
}

// WriteBool will write a boolean value as a single byte into w.
func WriteBool(w *bytes.Buffer, b bool) error {
	v := uint8(0)

	if b {
		v = 1
	}

	return WriteUint8(w, v)
}

// Read256 will read 32 bytes from r and return them as a slice of bytes.
func Read256(r *bytes.Buffer, b []byte) error {
	if len(b) != 32 {
		return errors.New("buffer for Read256 should be 32 bytes")
	}

	if _, err := r.Read(b); err != nil {
		return err
	}

	return nil
}

// Write256 will check the length of b and then write to w.
func Write256(w *bytes.Buffer, b []byte) error {
	if len(b) != 32 {
		return fmt.Errorf("b is not proper size - expected 32 bytes, is actually %d bytes", len(b))
	}

	_, err := w.Write(b)
	return err
}

// Read512 will read 64 bytes from r and return them as a slice of bytes.
func Read512(r *bytes.Buffer, b []byte) error {
	if len(b) != 64 {
		return errors.New("buffer for Read512 should be 64 bytes")
	}

	if _, err := r.Read(b); err != nil {
		return err
	}

	return nil
}

// Write512 will check the length of b and then write to w.
func Write512(w *bytes.Buffer, b []byte) error {
	if len(b) != 64 {
		return fmt.Errorf("b is not proper size - expected 64 bytes, is actually %d bytes", len(b))
	}

	_, err := w.Write(b)
	return err
}

// ReadBLS will read a compressed bls signature (33 bytes) from r and return it as a slice of bytes.
func ReadBLS(r *bytes.Buffer, b []byte) error {
	if len(b) != 33 {
		return errors.New("buffer for ReadBLS should be 33 bytes")
	}

	if _, err := r.Read(b); err != nil {
		return err
	}

	return nil
}

// WriteBLS will write a compressed bls signature (33 bytes) to w.
func WriteBLS(w *bytes.Buffer, b []byte) error {
	if len(b) != 33 {
		return fmt.Errorf("b is not proper size - expected 33 bytes, is actually %d bytes", len(b))
	}

	_, err := w.Write(b)
	return err
}

// ReadBLSPKey will read a compressed bls public key (96 bytes) from r and return it as a slice of bytes.
func ReadBLSPKey(r *bytes.Buffer, b []byte) error {
	if len(b) != 96 {
		return errors.New("buffer for ReadBLSPKey should be 96 bytes")
	}

	if _, err := r.Read(b); err != nil {
		return err
	}

	return nil
}

// WriteBLSPKey will write a compressed bls public key (96 bytes) to w.
func WriteBLSPKey(w *bytes.Buffer, b []byte) error {
	if len(b) != 96 {
		return fmt.Errorf("b is not proper size - expected 96 bytes, is actually %d bytes", len(b))
	}

	_, err := w.Write(b)
	return err
}
