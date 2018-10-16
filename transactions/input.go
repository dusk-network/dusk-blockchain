package transactions

import (
	"github.com/toghrulmaharramov/dusk-go/encoding"
	"io"
)

type Input struct {
	KeyImage  []byte // 32 bytes
	TxID      []byte // 32 bytes
	Index     uint8  // 1 byte
	Signature []byte // ~2500 bytes
}

func (i *Input) Encode(w io.Writer) error {
	if err := encoding.WriteHash(w, i.KeyImage); err != nil {
		return err
	}

	if err := encoding.WriteHash(w, i.TxID); err != nil {
		return err
	}

	if err := intSerializer.PutUint8(w, i.Index); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, i.Signature); err != nil {
		return err
	}

	return nil
}
