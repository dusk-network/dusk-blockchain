// Serialization functions for a variety of data structures, such as hashes,
// signatures, and booleans.

package encoding

import (
	"fmt"
	"io"
)

// ReadBool will read a single byte from r, turn it into a bool
// and pass it into v.
func ReadBool(r io.Reader, b *bool) error {
	var v uint8
	if err := ReadUint8(r, &v); err != nil {
		return err
	}
	*b = v != 0
	return nil
}

// WriteBool will write a boolean value as a single byte into w.
func WriteBool(w io.Writer, b bool) error {
	v := uint8(0)
	if b {
		v = 1
	}
	return WriteUint8(w, v)
}

// Read256 will read 32 bytes from r into b.
func Read256(r io.Reader, b *[]byte) error {
	*b = make([]byte, 32)
	n, err := r.Read(*b)
	if err != nil || n != len(*b) {
		return fmt.Errorf("encoding: Read256 read %v/32 bytes - %v", n, err)
	}
	return nil
}

// Write256 will check the length of b and then write to w.
func Write256(w io.Writer, b []byte) error {
	if len(b) != 32 {
		return fmt.Errorf("b is not proper size - expected 32 bytes, is actually %d bytes", len(b))
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

// Read512 will read 64 bytes from r into b.
func Read512(r io.Reader, b *[]byte) error {
	*b = make([]byte, 64)
	n, err := r.Read(*b)
	if err != nil || n != len(*b) {
		return fmt.Errorf("encoding: Read512 read %v/64 bytes - %v", n, err)
	}
	return nil
}

// Write512 will check the length of b and then write to w.
func Write512(w io.Writer, b []byte) error {
	if len(b) != 64 {
		return fmt.Errorf("b is not proper size - expected 64 bytes, is actually %d bytes", len(b))
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

// ReadBLS will read a compressed bls signature (33 bytes) from r into b.
func ReadBLS(r io.Reader, b *[]byte) error {
	*b = make([]byte, 33)
	n, err := r.Read(*b)
	if err != nil || n != len(*b) {
		return fmt.Errorf("encoding: ReadBLS read %v/33 bytes - %v", n, err)
	}
	return nil
}

// WriteBLS will write a compressed bls signature (33 bytes) to w.
func WriteBLS(w io.Writer, b []byte) error {
	if len(b) != 33 {
		return fmt.Errorf("b is not proper size - expected 33 bytes, is actually %d bytes", len(b))
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}
