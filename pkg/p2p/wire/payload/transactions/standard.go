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

// Decode a Standard struct from r.
// Implements TypeInfo interface.
func (s *Standard) Decode(r io.Reader) error {
	lInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.Inputs = make([]*Input, lInputs)
	for i := uint64(0); i < lInputs; i++ {
		s.Inputs[i] = &Input{}
		if err := s.Inputs[i].Decode(r); err != nil {
			return err
		}
	}

	lOutputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.Outputs = make([]*Output, lOutputs)
	for i := uint64(0); i < lOutputs; i++ {
		s.Outputs[i] = &Output{}
		if err := s.Outputs[i].Decode(r); err != nil {
			return err
		}
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &s.Fee); err != nil {
		return err
	}

	return nil
}

// Type returns the associated TxType for the Standard struct.
// Implements TypeInfo interface.
func (s *Standard) Type() TxType {
	return StandardType
}
