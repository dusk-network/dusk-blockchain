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

// intList is a simple free list of buffers that can be used by integer
// serialization functions, converting them to and from their binary encoding.
// This approach allows for concurrent serialization to happen without having
// a lot of memory allocations going on at the same time. Slices of bytes
// must be borrowed and returned with a length of 8 bytes.
type intList chan []byte

// Amount of buffers we can fit in the list. At maximum capacity this list will take up
// 8 * 1024 = 8192 bytes or 8KB of memory.
const intListCap = 1024

// Declare an intList with a length of intListCap
var IntSerializer intList = make(chan []byte, intListCap)

// Borrow returns a buffer of length 8.
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
func (l intList) Uint8(r io.Reader) (uint8, error) {
	b := l.Borrow()[:1]
	defer l.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		l.Return(b)
		return 0, err
	}
	return b[0], nil
}

// Read two bytes
func (l intList) Uint16(r io.Reader, o binary.ByteOrder) (uint16, error) {
	b := l.Borrow()[:2]
	defer l.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		l.Return(b)
		return 0, err
	}
	rv := o.Uint16(b)
	return rv, nil
}

// Read four bytes
func (l intList) Uint32(r io.Reader, o binary.ByteOrder) (uint32, error) {
	b := l.Borrow()[:4]
	defer l.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		l.Return(b)
		return 0, err
	}
	rv := o.Uint32(b)
	return rv, nil
}

// Read eight bytes
func (l intList) Uint64(r io.Reader, o binary.ByteOrder) (uint64, error) {
	b := l.Borrow()[:8]
	defer l.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		l.Return(b)
		return 0, err
	}
	rv := o.Uint64(b)
	return rv, nil
}

// Write single byte
func (l intList) PutUint8(w io.Writer, v uint8) error {
	b := l.Borrow()[:1]
	defer l.Return(b)
	b[0] = v
	_, err := w.Write(b)
	return err
}

// Write two bytes
func (l intList) PutUint16(w io.Writer, o binary.ByteOrder, v uint16) error {
	b := l.Borrow()[:2]
	defer l.Return(b)
	o.PutUint16(b, v)
	_, err := w.Write(b)
	return err
}

// Write four bytes
func (l intList) PutUint32(w io.Writer, o binary.ByteOrder, v uint32) error {
	b := l.Borrow()[:4]
	defer l.Return(b)
	o.PutUint32(b, v)
	_, err := w.Write(b)
	return err
}

// Write eight bytes
func (l intList) PutUint64(w io.Writer, o binary.ByteOrder, v uint64) error {
	b := l.Borrow()[:8]
	defer l.Return(b)
	o.PutUint64(b, v)
	_, err := w.Write(b)
	return err
}

// CompactSize functions

// ReadVarInt reads the discriminator byte of a CompactSize int,
// and then deserializes the number accordingly.
func ReadVarInt(r io.Reader) (uint64, error) {
	// Get discriminant from variable int
	d, err := intSerializer.Uint8(r)
	if err != nil {
		return 0, err
	}

	var rv uint64
	switch d {
	case 0xff:
		v, err := intSerializer.Uint64(r, le)
		if err != nil {
			return 0, err
		}
		rv = v

		// Canonical encoding check
		if m := uint64(0x100000000); rv < m {
			return 0, fmt.Errorf("ReadCompact() : non-canonical encoding")
		}
	case 0xfe:
		v, err := intSerializer.Uint32(r, le)
		if err != nil {
			return 0, err
		}
		rv = uint64(v)

		// Canonical encoding check
		if m := uint64(0x10000); rv < m {
			return 0, fmt.Errorf("ReadCompact() : non-canonical encoding")
		}
	case 0xfd:
		v, err := intSerializer.Uint16(r, le)
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
		return intSerializer.PutUint8(w, uint8(v))
	}

	if v <= 1<<16-1 {
		if err := intSerializer.PutUint8(w, 0xfd); err != nil {
			return err
		}
		return intSerializer.PutUint16(w, le, uint16(v))
	}

	if v <= 1<<32-1 {
		if err := intSerializer.PutUint8(w, 0xfe); err != nil {
			return err
		}
		return intSerializer.PutUint32(w, le, uint32(v))
	}

	if err := intSerializer.PutUint8(w, 0xff); err != nil {
		return err
	}
	return intSerializer.PutUint64(w, le, v)
}
