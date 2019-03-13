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
	M        []byte
	Fee      uint64
}

// NewBid returns a Bid struct populated with the passed parameters.
func NewBid(timelock uint64, m []byte, fee uint64) *Bid {
	return &Bid{
		Timelock: timelock,
		M:        m,
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

// Encode a Bid struct and write it to to the passed io.Writer.
func (b *Bid) encode(w io.Writer) error {
	if err := encodeInputs(w, b.Inputs); err != nil {
		return err
	}

	if err := b.Output.Encode(w); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, b.Timelock); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.M); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, b.Fee); err != nil {
		return err
	}

	return nil
}

// Decode a Bid struct from r and return it.
func decodeBidTransaction(r io.Reader) (*Bid, error) {
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

	var m []byte
	if err := encoding.Read256(r, &m); err != nil {
		return nil, err
	}

	var fee uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &fee); err != nil {
		return nil, err
	}

	return &Bid{
		Inputs:   inputs,
		Output:   output,
		Timelock: timelock,
		M:        m,
		Fee:      fee,
	}, nil
}

// Type returns the associated TxType for the Bid struct.
// Implements TypeInfo interface.
func (b *Bid) Type() TxType {
	return BidType
}
