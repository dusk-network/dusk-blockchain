// Variable length data serialization functions

package encoding

import (
	"bytes"
)

// ReadVarBytes will read a CompactSize int denoting the length, then
// proceeds to read that amount of bytes from r into b.
func ReadVarBytes(r *bytes.Buffer) ([]byte, error) {
	c, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	b := make([]byte, c)
	if _, err := r.Read(b); err != nil {
		return nil, err
	}
	return b, nil
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

// Convenience functions for strings. They will point to the functions above and
// handle type conversion.

// ReadString reads the data with ReadVarBytes and returns it as a string
// by simple type conversion.
func ReadString(r *bytes.Buffer) (string, error) {
	b, err := ReadVarBytes(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteString will write string s as a slice of bytes through WriteVarBytes.
func WriteString(w *bytes.Buffer, s string) error {
	return WriteVarBytes(w, []byte(s))
}
