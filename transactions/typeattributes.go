package transactions

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
)

// TypeAttributes holds the inputs, outputs and the public key associated
// with a stealth transaction.
type TypeAttributes struct {
	Inputs   []Input  //  m * 2565 bytes
	TxPubKey []byte   // 32 bytes
	Outputs  []Output // n * 40 bytes
}

// Encode will serialize a TypeAttributes struct of a stealthtx to w in byte format.
func (t *TypeAttributes) Encode(w io.Writer) error {
	// Inputs
	lenIn := uint64(len(t.Inputs))
	if err := encoding.WriteVarInt(w, lenIn); err != nil {
		return err
	}

	for _, in := range t.Inputs {
		if err := in.Encode(w); err != nil {
			return err
		}
	}

	// TxPubKey
	if err := encoding.WriteHash(w, t.TxPubKey); err != nil {
		return err
	}

	// Outputs
	lenOut := uint64(len(t.Outputs))
	if err := encoding.WriteVarInt(w, lenOut); err != nil {
		return err
	}

	for _, out := range t.Outputs {
		if err := out.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

// Decode will deserialize a TypeAttributes struct from a serialized stealthtx
// and populate the passed TypeAttributes struct.
func (t *TypeAttributes) Decode(r io.Reader) error {
	// Inputs
	inputCount, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Fill Inputs array with empty Input objects to populate
	t.Inputs = make([]Input, inputCount)
	for i := uint64(0); i < inputCount; i++ {
		if err := t.Inputs[i].Decode(r); err != nil {
			return err
		}
	}

	// TxPubKey
	txPubKey, err := encoding.ReadHash(r)
	if err != nil {
		return err
	}
	t.TxPubKey = txPubKey

	// Outputs
	outputCount, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Fill Outputs array with empty Output objects to populate
	t.Outputs = make([]Output, outputCount)
	for i := uint64(0); i < outputCount; i++ {
		if err := t.Outputs[i].Decode(r); err != nil {
			return err
		}
	}

	return nil
}
