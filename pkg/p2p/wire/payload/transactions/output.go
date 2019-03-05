package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Output defines an output in a stealth transaction.
type Output struct {
	Amount uint64 // 8 bytes
	P      []byte // 32 bytes
	BP     []byte // Variable size
}

// NewOutput constructs a new Output from the passed parameters.
func NewOutput(amount uint64, dest []byte, proof []byte) *Output {
	return &Output{
		Amount: amount,
		P:      dest,
		BP:     proof,
	}
}

// Encode an Output object and write to w.
func (o *Output) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, o.Amount); err != nil {
		return err
	}

	if err := encoding.Write256(w, o.P); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, o.BP); err != nil {
		return err
	}

	return nil
}

// Decode an Output object from r into o.
func (o *Output) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &o.Amount); err != nil {
		return err
	}

	if err := encoding.Read256(r, &o.P); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &o.BP); err != nil {
		return err
	}

	return nil
}
