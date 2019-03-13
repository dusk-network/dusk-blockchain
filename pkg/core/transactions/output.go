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

// Decode an Output object from r and return it.
func decodeOutput(r io.Reader) (*Output, error) {
	var amount uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &amount); err != nil {
		return nil, err
	}

	var p []byte
	if err := encoding.Read256(r, &p); err != nil {
		return nil, err
	}

	var bp []byte
	if err := encoding.ReadVarBytes(r, &bp); err != nil {
		return nil, err
	}

	return &Output{
		Amount: amount,
		P:      p,
		BP:     bp,
	}, nil
}

func decodeOutputs(r io.Reader) ([]*Output, error) {
	lOutputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	outputs := make([]*Output, lOutputs)
	for i := uint64(0); i < lOutputs; i++ {
		output, err := decodeOutput(r)
		if err != nil {
			return nil, err
		}

		outputs[i] = output
	}

	return outputs, nil
}
