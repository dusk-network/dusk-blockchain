package transactions

import (
	"encoding/binary"
	"io"
)

func (t *TypeAttributes) Encode(w io.Writer) error {

	//Inputs
	lenIn := len(t.Inputs)
	err := binary.Write(w, binary.LittleEndian, uint64(lenIn)) // replace with varuint
	if err != nil {
		return err
	}

	for _, in := range t.Inputs {
		err = in.Encode(w)
		if err != nil {
			return err
		}
	}

	err = binary.Write(w, binary.LittleEndian, t.TxPubKey)
	if err != nil {
		return err
	}
	lenOut := len(t.Outputs)
	err = binary.Write(w, binary.LittleEndian, uint64(lenOut)) // replace with varuint
	for _, out := range t.Outputs {
		err = out.Encode(w)
		if err != nil {
			return err
		}
	}
	return err
}
