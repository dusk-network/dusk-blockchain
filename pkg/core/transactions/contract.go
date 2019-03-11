package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Contract defines the transaction info for a contract transaction
type Contract struct {
	Inputs   []*Input
	Outputs  []*Output
	Timelock uint64
	Fee      uint64
}

// NewContract will return a Contract struct with the passed parameters.
func NewContract(timelock, fee uint64) *Contract {
	return &Contract{
		Timelock: timelock,
		Fee:      fee,
	}
}

// AddInput will add an input to the Inputs array of the Contract struct.
func (c *Contract) AddInput(input *Input) {
	c.Inputs = append(c.Inputs, input)
}

// AddOutput will add an output to the Outputs array of the Contract struct.
func (c *Contract) AddOutput(output *Output) {
	c.Outputs = append(c.Outputs, output)
}

// Encode a Contract struct and write to w.
// Implements TypeInfo interface.
func (c *Contract) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(c.Inputs))); err != nil {
		return err
	}

	for _, input := range c.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}
	if err := encoding.WriteVarInt(w, uint64(len(c.Outputs))); err != nil {
		return err
	}

	for _, output := range c.Outputs {
		if err := output.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, c.Timelock); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, c.Fee); err != nil {
		return err
	}

	return nil
}

// Decode a Contract struct from r and return it.
func decodeContractTransaction(r io.Reader) (*Contract, error) {
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

	return &Contract{
		Inputs:   inputs,
		Outputs:  outputs,
		Timelock: timelock,
		Fee:      fee,
	}, nil
}

// Type returns the associated TxType for the Contract struct.
// Implements TypeInfo interface.
func (c *Contract) Type() TxType {
	return ContractType
}
