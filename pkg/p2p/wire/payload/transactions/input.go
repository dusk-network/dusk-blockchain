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

// Encode an Input object and write to w.
func (i *Input) Encode(w io.Writer) error {
	if err := encoding.Write256(w, i.KeyImage); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.TxID); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, i.Index); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, i.Signature); err != nil {
		return err
	}

	return nil
}

// Decode an Input object from r into i.
func (i *Input) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &i.KeyImage); err != nil {
		return err
	}

	if err := encoding.Read256(r, &i.TxID); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &i.Index); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &i.Signature); err != nil {
		return err
	}

	return nil
}
