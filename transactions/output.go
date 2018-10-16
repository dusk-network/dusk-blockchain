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

// Encode will serialize an Output struct to w in byte format.
func (o *Output) Encode(w io.Writer) error {
	if err := encoding.PutUint64(w, binary.LittleEndian, o.Amount); err != nil {
		return err
	}

	if err := encoding.WriteHash(w, o.P); err != nil {
		return err
	}

	return nil
}

// Decode will deserialize an Output struct and populate the passed Output struct
// with it's details.
func (o *Output) Decode(r io.Reader) error {
	amount, err := encoding.Uint64(r, binary.LittleEndian)
	if err != nil {
		return err
	}
	o.Amount = amount

	P, err := encoding.ReadHash(r)
	if err != nil {
		return err
	}
	o.P = P

	return nil
}
