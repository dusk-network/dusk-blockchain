package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Stake defines the transaction info for a staking transaction.
type Stake struct {
	Inputs []*Input
	*Output
	Timelock  uint64
	Fee       uint64
	PubKeyEd  []byte
	PubKeyBLS []byte
}

// NewStake will return a Stake struct with the passed public keys.
func NewStake(timelock, fee uint64, pubKeyEd, pubKeyBLS []byte) *Stake {
	return &Stake{
		Timelock:  timelock,
		Fee:       fee,
		PubKeyEd:  pubKeyEd,
		PubKeyBLS: pubKeyBLS,
	}
}

// AddInput will add an input to the Inputs array of the Stake struct.
func (s *Stake) AddInput(input *Input) {
	s.Inputs = append(s.Inputs, input)
}

// AddOutput will populate the Stake struct's Output field.
func (s *Stake) AddOutput(output *Output) {
	s.Output = output
}

// Encode a Stake struct and write to w.
// Implements TypeInfo interface.
func (s *Stake) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(s.Inputs))); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}

	if err := s.Output.Encode(w); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, s.Timelock); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, s.Fee); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a Stake struct from r and return it.
func decodeStakeTransaction(r io.Reader) (*Stake, error) {
	inputs, err := decodeInputs(r)
	if err != nil {
		return nil, err
	}

	output, err := decodeOutput(r)
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

	var pubKeyEd []byte
	if err := encoding.Read256(r, &pubKeyEd); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, err
	}

	return &Stake{
		Inputs:    inputs,
		Output:    output,
		Timelock:  timelock,
		Fee:       fee,
		PubKeyEd:  pubKeyEd,
		PubKeyBLS: pubKeyBLS,
	}, nil
}

// Type returns the associated TxType for the Stake struct.
// Implements TypeInfo interface.
func (s *Stake) Type() TxType {
	return StakeType
}
