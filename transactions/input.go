package transactions

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
)

// Input defines an input in a stealth transaction.
type Input struct {
	KeyImage  []byte // 32 bytes
	TxID      []byte // 32 bytes
	Index     uint8  // 1 byte
	Signature []byte // ~2500 bytes
}

// Encode will serialize an Input struct to w in byte format.
func (i *Input) Encode(w io.Writer) error {
	// KeyImage
	if err := encoding.Write256(w, i.KeyImage); err != nil {
		return err
	}

	// TxID
	if err := encoding.Write256(w, i.TxID); err != nil {
		return err
	}

	// Index
	if err := encoding.WriteUint8(w, i.Index); err != nil {
		return err
	}

	// Signature
	if err := encoding.WriteVarBytes(w, i.Signature); err != nil {
		return err
	}

	return nil
}

// Decode will deserialize an Input struct and populate the passed Input struct
// with it's details.
func (i *Input) Decode(r io.Reader) error {
	// KeyImage
	if err := encoding.Read256(r, &i.KeyImage); err != nil {
		return err
	}

	// TxID
	if err := encoding.Read256(r, &i.TxID); err != nil {
		return err
	}

	// Index
	if err := encoding.ReadUint8(r, &i.Index); err != nil {
		return err
	}

	// Signature
	if err := encoding.ReadVarBytes(r, &i.Signature); err != nil {
		return err
	}

	return nil
}
