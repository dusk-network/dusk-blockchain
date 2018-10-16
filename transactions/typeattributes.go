package transactions

import (
	"github.com/toghrulmaharramov/dusk-go/encoding"
	"io"
)

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

	if err := encoding.WriteHash(w, t.TxPubKey); err != nil {
		return err
	}

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
