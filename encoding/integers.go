// Serialization functions for integers and CompactSize integers
package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
)

var (
	// Convenience variable
	le = binary.LittleEndian
)

// Set up a free list of buffers for use by the functions in this source file.
// This approach will reduce the amount of memory allocations when the software is
// serializing/deserializing integers and will help to increase performance.
type intList chan []byte

// Amount of buffers we can fit in the list. At maximum capacity this list will take up
// 8 * 1024 = 8192 bytes or 8KB of memory.
const intListCap = 1024

// Declare an intList with a length of intListCap.
var intSerializer intList = make(chan []byte, intListCap)

// Borrow returns a buffer of length 8. If none are available, allocate one.
func (l intList) Borrow() []byte {
	var b []byte
	select {
	case b = <-l:
	default:
		b = make([]byte, 8)
	}
	return b[:8]
}

// Return a byte buffer back to the intList. If the buffer does not have a size of 8 bytes,
// the byte slice is discarded and the garbage collector will deal with it.
// The same will happen if the intList is full.
func (l intList) Return(b []byte) {
	// Ignore buffers with unexpected size
	if cap(b) != 8 {
		return // Goes to the garbage collector
	}
	select {
	case l <- b:
	default: // If full, goes to the garbage collector
	}
}

// Base integer serialization functions

// Read single byte
func Uint8(r io.Reader) (uint8, error) {
	b := intSerializer.Borrow()[:1]
	defer intSerializer.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	return b[0], nil
}

// Read two bytes
func Uint16(r io.Reader, o binary.ByteOrder) (uint16, error) {
	b := intSerializer.Borrow()[:2]
	defer intSerializer.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	rv := o.Uint16(b)
	return rv, nil
}

// Read four bytes
func Uint32(r io.Reader, o binary.ByteOrder) (uint32, error) {
	b := intSerializer.Borrow()[:4]
	defer intSerializer.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	rv := o.Uint32(b)
	return rv, nil
}

// Read eight bytes
func Uint64(r io.Reader, o binary.ByteOrder) (uint64, error) {
	b := intSerializer.Borrow()[:8]
	defer intSerializer.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	rv := o.Uint64(b)
	return rv, nil
}

// Write single byte
func PutUint8(w io.Writer, v uint8) error {
	b := intSerializer.Borrow()[:1]
	defer intSerializer.Return(b)
	b[0] = v
	_, err := w.Write(b)
	return err
}

// Write two bytes
func PutUint16(w io.Writer, o binary.ByteOrder, v uint16) error {
	b := intSerializer.Borrow()[:2]
	defer intSerializer.Return(b)
	o.PutUint16(b, v)
	_, err := w.Write(b)
	return err
}

// Write four bytes
func PutUint32(w io.Writer, o binary.ByteOrder, v uint32) error {
	b := intSerializer.Borrow()[:4]
	defer intSerializer.Return(b)
	o.PutUint32(b, v)
	_, err := w.Write(b)
	return err
}

// Write eight bytes
func PutUint64(w io.Writer, o binary.ByteOrder, v uint64) error {
	b := intSerializer.Borrow()[:8]
	defer intSerializer.Return(b)
	o.PutUint64(b, v)
	_, err := w.Write(b)
	return err
}

// CompactSize functions

// ReadVarInt reads the discriminator byte of a CompactSize int,
// and then deserializes the number accordingly.
func ReadVarInt(r io.Reader) (uint64, error) {
	// Get discriminant from variable int
	d, err := Uint8(r)
	if err != nil {
		return 0, err
	}

	var rv uint64
	switch d {
	case 0xff:
		v, err := Uint64(r, le)
		if err != nil {
			return 0, err
		}
		rv = v

		// Canonical encoding check
		if m := uint64(0x100000000); rv < m {
			return 0, fmt.Errorf("ReadCompact() : non-canonical encoding")
		}
	case 0xfe:
		v, err := Uint32(r, le)
		if err != nil {
			return 0, err
		}
		rv = uint64(v)

		// Canonical encoding check
		if m := uint64(0x10000); rv < m {
			return 0, fmt.Errorf("ReadCompact() : non-canonical encoding")
		}
	case 0xfd:
		v, err := Uint16(r, le)
		if err != nil {
			return 0, err
		}
		rv = uint64(v)

		// Canonical encoding check
		if m := uint64(0xfd); rv < m {
			return 0, fmt.Errorf("ReadCompact() : non-canonical encoding")
		}
	default:
		rv = uint64(d)
	}

	return rv, nil
}

// WriteVarInt writes a CompactSize integer with a number of bytes depending on it's value
func WriteVarInt(w io.Writer, v uint64) error {
	if v < 0xfd {
		return PutUint8(w, uint8(v))
	}

	if v <= 1<<16-1 {
		if err := PutUint8(w, 0xfd); err != nil {
			return err
		}
		return PutUint16(w, le, uint16(v))
	}

	if v <= 1<<32-1 {
		if err := PutUint8(w, 0xfe); err != nil {
			return err
		}
		return PutUint32(w, le, uint32(v))
	}

	if err := PutUint8(w, 0xff); err != nil {
		return err
	}
	return PutUint64(w, le, v)
}

// VarIntSerializeSize returns the number of bytes needed to serialize a CompactSize int
// the size of v
func VarIntSerializeSize(v uint64) int {
	// Small enough to write in 1 byte (uint8)
	if v < 0xfd {
		return 1
	}

	// Discriminant byte plus 2 (uint16)
	if v <= 1<<16-1 {
		return 3
	}

	// Discriminant byte plus 4 (uint32)
	if v <= 1<<32-1 {
		return 5
	}

	// Discriminant byte plus 8 (uint64)
	return 9
}
