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
	// PubKey is the one-time public key that this output was created with.
	// It acts as a output identifer because they can only be used once
	PubKey []byte // 32 bytes
	// PseudoCommitment acts as the dummy commitment used to construct the
	// intermediate ring signatures
	PseudoCommitment []byte // 32 bytes
	// Signature is the ring signature that is used
	// to sign the transaction
	Signature []byte // ~2500 bytes
}

// NewInput constructs a new Input from the passed parameters.
func NewInput(keyImage []byte, pubKey, pseudoComm, sig []byte) (*Input, error) {

	if len(keyImage) != 32 {
		return nil, errors.New("key image does not equal 32 bytes")
	}

	if len(pubKey) != 32 {
		return nil, errors.New("pubKey does not equal 32 bytes")
	}

	if len(pseudoComm) != 32 {
		return nil, errors.New("Pseudo Commitment  does not equal 32 bytes")
	}

	return &Input{
		KeyImage:         keyImage,
		PubKey:           pubKey,
		PseudoCommitment: pseudoComm,
		Signature:        sig,
	}, nil
}

// Encode an Input object into an io.Writer.
func (i *Input) Encode(w io.Writer) error {
	if err := encoding.Write256(w, i.KeyImage); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.PubKey); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.PseudoCommitment); err != nil {
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

	if err := encoding.Read256(r, &i.PubKey); err != nil {
		return err
	}

	if err := encoding.Read256(r, &i.PseudoCommitment); err != nil {
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

	if !bytes.Equal(i.PubKey, in.PubKey) {
		return false
	}

	if !bytes.Equal(i.PseudoCommitment, in.PseudoCommitment) {
		return false
	}

	return bytes.Equal(i.Signature, in.Signature)
}
