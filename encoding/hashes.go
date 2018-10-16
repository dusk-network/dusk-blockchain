// Hash serialization functions
package encoding

import (
	"fmt"
	"io"
)

// hashList is a simple free list of buffers that can be used for deserializing hashes
// in a more efficient way.
type hashList chan []byte

// Amount of buffers we can fit in the list. At maximum capacity this list will take up
// 8 * 1024 = 8192 bytes or 8KB of memory.
const hashListCap = 1024

// Declare a hashList with a length of hashListCap
var HashSerializer hashList = make(chan []byte, hashListCap)

// Borrow a 32 byte buffer from the hashList. If none are available, allocate one.
func (l hashList) Borrow() []byte {
	var b []byte
	select {
	case b = <-l:
	default:
		b = make([]byte, 32)
	}
	return b[:32]
}

// Return a byte buffer to the hashList. If it doesn't meet the size requirement,
// the buffer will be left for the garbage collector to be cleaned up. The same will
// happen if the list is full.
func (l hashList) Return(b []byte) {
	// Ignore buffers with unexpected size
	if cap(b) != 32 {
		return // Goes to the garbage collector
	}
	select {
	case l <- b:
	default: // If full, goes to the garbage collector
	}
}

// Hash deserialization function. Hashes are stored as a slice of 32 bytes, so we can
// use the free list to get it out.
func (l hashList) ReadHash(r io.Reader) ([]byte, error) {
	b := l.Borrow()[:32]
	defer l.Return(b)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

// WriteHash will check the hash length and then write the data to w. If an error
// is encountered, return it
func WriteHash(w io.Writer, hash []byte) error {
	if len(hash) != 32 {
		return fmt.Errorf("hash is not proper size - expected 32 bytes, is actually %d bytes", len(hash))
	}
	if _, err := w.Write(hash); err != nil {
		return err
	}
	return nil
}