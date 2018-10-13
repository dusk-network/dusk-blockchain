package transactions

import (
	"encoding/binary"
	"github.com/toghrulmaharramov/dusk-go/encoding"
	"io"
)

type Output struct {
	Amount uint64 // 8 bytes
	P      []byte // 32 bytes
}

func (o *Output) Encode(w io.Writer) error {
	if err := intSerializer.PutUint64(w, binary.LittleEndian, o.Amount); err != nil {
		return err
	}

	if err := encoding.WriteHash(w, o.P); err != nil {
		return err
	}

	return nil
}
