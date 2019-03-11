package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Timelock defines the transaction info for a time-locked transaction
type Timelock struct {
	Inputs   []*Input
	Outputs  []*Output
	Timelock uint64
	Fee      uint64
}

// NewTimelock will return a Timelock struct with the passed parameters.
func NewTimelock(timelock, fee uint64) *Timelock {
	return &Timelock{
		Timelock: timelock,
		Fee:      fee,
	}
}

// AddInput will add an input to the Inputs array of the Timelock struct.
func (t *Timelock) AddInput(input *Input) {
	t.Inputs = append(t.Inputs, input)
}

// AddOutput will add an output to the Outputs array of the Timelock struct.
func (t *Timelock) AddOutput(output *Output) {
	t.Outputs = append(t.Outputs, output)
}

// Encode a Timelock struct and write to w.
// Implements TypeInfo interface.
func (t *Timelock) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(t.Inputs))); err != nil {
		return err
	}

	for _, input := range t.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}
	if err := encoding.WriteVarInt(w, uint64(len(t.Outputs))); err != nil {
		return err
	}

	for _, output := range t.Outputs {
		if err := output.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, t.Timelock); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, t.Fee); err != nil {
		return err
	}

	return nil
}

// Decode a Timelock struct from r and return it.
func decodeTimelockTransaction(r io.Reader) (*Timelock, error) {
	inputs, err := decodeInputs(r)
	if err != nil {
		return nil, err
	}

	outputs, err := decodeOutputs(r)
	if err != nil {
		return nil, err
	}

	var timelock uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &timelock); err != nil {
		return nil, err
	}

	var fee uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &fee); err != nil {
		return nil, err
	}

	return &Timelock{
		Inputs:   inputs,
		Outputs:  outputs,
		Timelock: timelock,
		Fee:      fee,
	}, nil
}

// Type returns the associated TxType for the Timelock struct.
// Implements TypeInfo interface.
func (t *Timelock) Type() TxType {
	return TimelockType
}
