package transactions

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Bid defines the transaction information for a blind bid transaction.
type Bid struct {
	Inputs []*Input
	*Output
	Timelock uint64
	Secret   []byte
	Fee      uint64
}

// NewBid returns a Bid struct populated with the passed parameters.
func NewBid(timelock uint64, secret []byte, fee uint64) *Bid {
	return &Bid{
		Timelock: timelock,
		Secret:   secret,
		Fee:      fee,
	}
}

// AddInput will add an input to the Inputs array of the Bid struct.
func (b *Bid) AddInput(input *Input) {
	b.Inputs = append(b.Inputs, input)
}

// AddOutput will populate the Bid struct's Output field.
func (b *Bid) AddOutput(output *Output) {
	b.Output = output
}

// Encode a Bid struct and write to w.
// Implements TypeInfo interface.
func (b *Bid) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(b.Inputs))); err != nil {
		return err
	}

	for _, input := range b.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}

	if err := b.Output.Encode(w); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, b.Timelock); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.Secret); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, b.Fee); err != nil {
		return err
	}

	return nil
}

// Decode a Bid struct from r.
// Implements TypeInfo interface.
func (b *Bid) Decode(r io.Reader) error {
	lInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	b.Inputs = make([]*Input, lInputs)
	for i := uint64(0); i < lInputs; i++ {
		b.Inputs[i] = &Input{}
		if err := b.Inputs[i].Decode(r); err != nil {
			return err
		}
	}

	b.Output = &Output{}
	if err := b.Output.Decode(r); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &b.Timelock); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.Secret); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &b.Fee); err != nil {
		return err
	}

	return nil
}

// Type returns the associated TxType for the Bid struct.
// Implements TypeInfo interface.
func (b *Bid) Type() TxType {
	return BidType
}
