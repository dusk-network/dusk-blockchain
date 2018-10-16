// Variable string serialization methods
package encoding

import (
	"io"
)

func ReadVarString(r io.Reader) (string, error) {
	c, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}

	b := make([]byte, c)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func WriteVarString(w io.Writer, s string) error {
	if err := WriteVarInt(w, uint64(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}
