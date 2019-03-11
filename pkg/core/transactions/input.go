package transactions

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Input defines an input in a stealth transaction.
type Input struct {
	KeyImage  []byte // 32 bytes
	TxID      []byte // 32 bytes
	Index     uint8  // 1 byte
	Signature []byte // ~2500 bytes
}

// NewInput constructs a new Input from the passed parameters.
func NewInput(keyImage []byte, txID []byte, index uint8, sig []byte) *Input {
	return &Input{
		KeyImage:  keyImage,
		TxID:      txID,
		Index:     index,
		Signature: sig,
	}
}

// Encode an Input object to w.
func encodeInput(w io.Writer, input *Input) error {
	if err := encoding.Write256(w, input.KeyImage); err != nil {
		return err
	}

	if err := encoding.Write256(w, input.TxID); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, input.Index); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, input.Signature); err != nil {
		return err
	}

	return nil
}

func encodeInputs(w io.Writer, inputs []*Input) error {
	if err := encoding.WriteVarInt(w, uint64(len(inputs))); err != nil {
		return err
	}

	for _, input := range inputs {
		if err := encodeInput(w, input); err != nil {
			return err
		}
	}

	return nil
}

// Decode an Input object from r and return it.
func decodeInput(r io.Reader) (*Input, error) {
	var keyImage []byte
	if err := encoding.Read256(r, &keyImage); err != nil {
		return nil, err
	}

	var txID []byte
	if err := encoding.Read256(r, &txID); err != nil {
		return nil, err
	}

	var index uint8
	if err := encoding.ReadUint8(r, &index); err != nil {
		return nil, err
	}

	var signature []byte
	if err := encoding.ReadVarBytes(r, &signature); err != nil {
		return nil, err
	}

	return &Input{
		KeyImage:  keyImage,
		TxID:      txID,
		Index:     index,
		Signature: signature,
	}, nil
}

func decodeInputs(r io.Reader) ([]*Input, error) {
	lInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	inputs := make([]*Input, lInputs)
	for i := uint64(0); i < lInputs; i++ {
		input, err := decodeInput(r)
		if err != nil {
			return nil, err
		}

		inputs[i] = input
	}

	return inputs, nil
}
