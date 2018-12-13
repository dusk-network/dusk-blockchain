package transactions

import (
	"encoding/binary"
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
)

// Output defines an output in a stealth transaction.
type Output struct {
	Amount uint64 // 8 bytes
	P      []byte // 32 bytes
}

// Encode will serialize an Output struct to w in byte format.
func (o *Output) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, o.Amount); err != nil {
		return err
	}

	if err := encoding.Write256(w, o.P); err != nil {
		return err
	}

	return nil
}

// Decode will deserialize an Output struct and populate the passed Output struct
// with it's details.
func (o *Output) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &o.Amount); err != nil {
		return err
	}

	if err := encoding.Read256(r, &o.P); err != nil {
		return err
	}

	return nil
}
