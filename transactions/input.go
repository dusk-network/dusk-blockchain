package transactions

import (
	"encoding/binary"
	"io"
)

type Input struct {
	KeyImage  []byte // 32 bytes
	TxID      []byte // 32 bytes
	Index     uint8  // 1 byte
	Signature []byte // ~2500 bytes
}

func (i *Input) Encode(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, i.KeyImage)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, i.TxID)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, i.Index)
	if err != nil {
		return err
	}
	lenSig := len(i.Signature)
	err = binary.Write(w, binary.LittleEndian, uint64(lenSig)) // replace with varuint
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, i.Signature)
	return err
}
