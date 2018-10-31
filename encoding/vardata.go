// Variable length data serialization functions

package encoding

import "io"

// ReadVarBytes will read a CompactSize int denoting the length, then
// proceeds to read that amount of bytes into r and passes it back as a slice.
func ReadVarBytes(r io.Reader) ([]byte, error) {
	c, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	b := make([]byte, c)
	if _, err = io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
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
func ReadString(r io.Reader) (string, error) {
	b, err := ReadVarBytes(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteString will write string s as a slice of bytes through WriteVarBytes.
func WriteString(w io.Writer, s string) error {
	err := WriteVarBytes(w, []byte(s))
	return err
}
