package transactions

import (
	"encoding/binary"
	"io"
)

type Output struct {
	Amount uint64 // 8 bytes
	P      []byte // 32 bytes
}

func (o *Output) Encode(w io.Writer) error {

	err := binary.Write(w, binary.LittleEndian, o.Amount)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, o.P)
	return err
}
