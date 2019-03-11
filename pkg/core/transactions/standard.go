package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Standard defines the transaction info for a standard transaction
type Standard struct {
	Inputs  []*Input
	Outputs []*Output
	Fee     uint64
}

// NewStandard will return a Standard struct with the passed parameters.
func NewStandard(fee uint64) *Standard {
	return &Standard{
		Fee: fee,
	}
}

// AddInput will add an input to the Inputs array of the Standard struct.
func (s *Standard) AddInput(input *Input) {
	s.Inputs = append(s.Inputs, input)
}

// AddOutput will add an output to the Outputs array of the Standard struct.
func (s *Standard) AddOutput(output *Output) {
	s.Outputs = append(s.Outputs, output)
}

// Encode a Standard struct and write to w.
// Implements TypeInfo interface.
func (s *Standard) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(s.Inputs))); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}
	if err := encoding.WriteVarInt(w, uint64(len(s.Outputs))); err != nil {
		return err
	}

	for _, output := range s.Outputs {
		if err := output.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, s.Fee); err != nil {
		return err
	}

	return nil
}

// Decode a Standard struct from r and return it.
func decodeStandardTransaction(r io.Reader) (*Standard, error) {
	inputs, err := decodeInputs(r)
	if err != nil {
		return nil, err
	}

	outputs, err := decodeOutputs(r)
	if err != nil {
		return nil, err
	}

	var fee uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &fee); err != nil {
		return nil, err
	}

	return &Standard{
		Inputs:  inputs,
		Outputs: outputs,
		Fee:     fee,
	}, nil
}

// Type returns the associated TxType for the Standard struct.
// Implements TypeInfo interface.
func (s *Standard) Type() TxType {
	return StandardType
}
