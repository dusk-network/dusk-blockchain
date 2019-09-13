// Serialization functions for a variety of data structures, such as hashes,
// signatures, and booleans.

package encoding

import (
	"bytes"
	"fmt"
)

// ReadBool will read a single byte from r, turn it into a bool
// and return it.
func ReadBool(r *bytes.Buffer) (bool, error) {
	v, err := ReadUint8(r)
	if err != nil {
		return false, err
	}
	return v != 0, nil
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
func Read256(r *bytes.Buffer) ([]byte, error) {
	var b [32]byte
	if _, err := r.Read(b[:]); err != nil {
		return nil, err
	}
	return b[:], nil
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
func Read512(r *bytes.Buffer) ([]byte, error) {
	var b [64]byte
	if _, err := r.Read(b[:]); err != nil {
		return nil, err
	}
	return b[:], nil
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
func ReadBLS(r *bytes.Buffer) ([]byte, error) {
	var b [33]byte
	if _, err := r.Read(b[:]); err != nil {
		return nil, err
	}
	return b[:], nil
}

// WriteBLS will write a compressed bls signature (33 bytes) to w.
func WriteBLS(w *bytes.Buffer, b []byte) error {
	if len(b) != 33 {
		return fmt.Errorf("b is not proper size - expected 33 bytes, is actually %d bytes", len(b))
	}
	_, err := w.Write(b)
	return err
}
