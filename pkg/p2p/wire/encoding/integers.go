// Serialization functions for integers

package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadUint8 will read a single byte and put it into v.
func ReadUint8(r io.Reader, v *uint8) error {
	var b [1]byte
	if _, err := r.Read(b[:]); err != nil {
		return err
	}
	// Since we only read one byte, it will either succeed
	// or throw an EOF error. Thus, we don't check the number of
	// bytes read here.
	*v = b[0]
	return nil
}

// ReadUint16 will read two bytes and convert them to a uint16
// from the specified byte order. The result is put into v.
func ReadUint16(r io.Reader, o binary.ByteOrder, v *uint16) error {
	var b [2]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return fmt.Errorf("encoding: ReadUint16 read %v/2 bytes - %v", n, err)
	}
	*v = o.Uint16(b[:])
	return nil
}

// ReadUint32 will read four bytes and convert them to a uint32
// from the specified byte order. The result is put into v.
func ReadUint32(r io.Reader, o binary.ByteOrder, v *uint32) error {
	var b [4]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return fmt.Errorf("encoding: ReadUint32 read %v/4 bytes - %v", n, err)
	}
	*v = o.Uint32(b[:])
	return nil
}

// ReadUint64 will read eight bytes and convert them to a uint64
// from the specified byte order. The result is put into v.
func ReadUint64(r io.Reader, o binary.ByteOrder, v *uint64) error {
	var b [8]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return fmt.Errorf("encoding: ReadUint64 read %v/8 bytes - %v", n, err)
	}
	*v = o.Uint64(b[:])
	return nil
}

// WriteUint8 will write a single byte.
func WriteUint8(w io.Writer, v uint8) error {
	var b [1]byte
	b[0] = v
	_, err := w.Write(b[:])
	return err
}

// WriteUint16 will write two bytes in the specified byte order.
func WriteUint16(w io.Writer, o binary.ByteOrder, v uint16) error {
	var b [2]byte
	o.PutUint16(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// WriteUint32 will write four bytes in the specified byte order.
func WriteUint32(w io.Writer, o binary.ByteOrder, v uint32) error {
	var b [4]byte
	o.PutUint32(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// WriteUint64 will write eight bytes in the specified byte order.
func WriteUint64(w io.Writer, o binary.ByteOrder, v uint64) error {
	var b [8]byte
	o.PutUint64(b[:], v)
	_, err := w.Write(b[:])
	return err
}
