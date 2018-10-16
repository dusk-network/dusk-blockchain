// Variable length byte data serialization functions
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