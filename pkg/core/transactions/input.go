package transactions

import (
	"bytes"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Input defines an input in a standard transaction.
type Input struct {
	// KeyImage is the image of the key that is being used to
	// sign the transaction
	KeyImage []byte // 32 bytes
	// TxID is the transaction identifier of the transaction
	// to which this input was an output.
	TxID []byte // 32 bytes
	// Index is the position in the current transaction
	// that this input is placed at. Index 0 signifies the coinbase transaction.
	Index uint8 // 1 byte
	// Signature is the ring signature that is used
	// to sign the transaction
	Signature []byte // ~2500 bytes
}

// NewInput constructs a new Input from the passed parameters.
func NewInput(keyImage []byte, txID []byte, index uint8, sig []byte) (*Input, error) {

	if len(keyImage) != 32 {
		return nil, errors.New("key image does not equal 32 bytes")
	}

	if len(txID) != 32 {
		return nil, errors.New("txID does not equal 32 bytes")
	}

	return &Input{
		KeyImage:  keyImage,
		TxID:      txID,
		Index:     index,
		Signature: sig,
	}, nil
}

// Encode an Input object into an io.Writer.
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

// Decode an Input object from a io.reader.
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

// Equals returns true if two inputs are the same
func (i *Input) Equals(in *Input) bool {
	if in == nil || i == nil {
		return false
	}

	if !bytes.Equal(i.KeyImage, in.KeyImage) {
		return false
	}

	if !bytes.Equal(i.TxID, in.TxID) {
		return false
	}

	if !bytes.Equal(i.Signature, in.Signature) {
		return false
	}

	// Omit Index equality; same input could be at two different
	// places in a tx

	return true
}
