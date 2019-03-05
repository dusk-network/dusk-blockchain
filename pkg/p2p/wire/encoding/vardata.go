// Variable length data serialization functions

package encoding

import (
	"fmt"
	"io"
)

// ReadVarBytes will read a CompactSize int denoting the length, then
// proceeds to read that amount of bytes from r into b.
func ReadVarBytes(r io.Reader, b *[]byte) error {
	c, err := ReadVarInt(r)
	if err != nil {
		return err
	}

	*b = make([]byte, c)
	n, err := r.Read(*b)
	if err != nil || n != len(*b) {
		return fmt.Errorf("encoding: ReadVarBytes read %v/%v bytes - %v", n, c, err)
	}
	return nil
}

// WriteVarBytes will serialize a CompactSize int denoting the length, then
// proceeds to write b into w.
func WriteVarBytes(w io.Writer, b []byte) error {
	if err := WriteVarInt(w, uint64(len(b))); err != nil {
		return err
	}

	_, err := w.Write(b)
	return err
}

// Convenience functions for strings. They will point to the functions above and
// handle type conversion.

// ReadString reads the data with ReadVarBytes and returns it as a string
// by simple type conversion.
func ReadString(r io.Reader, s *string) error {
	var b []byte
	if err := ReadVarBytes(r, &b); err != nil {
		return err
	}
	*s = string(b)
	return nil
}

// WriteString will write string s as a slice of bytes through WriteVarBytes.
func WriteString(w io.Writer, s string) error {
	return WriteVarBytes(w, []byte(s))
}
