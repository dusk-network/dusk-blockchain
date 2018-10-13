package transactions

import (
	"encoding/binary"
	"io"
)

type Stealth struct {
	Version uint8          // 1 byte
	Type    uint8          // 1 byte
	R       []byte         // 32 bytes
	TA      TypeAttributes // (m * 2565) + 32 + (n * 40) m = # inputs, n = # of outputs
}

type TypeAttributes struct {
	Inputs   []Input  //  m * 2565 bytes
	TxPubKey []byte   // 32 bytes
	Outputs  []Output // n * 40 bytes
}

func (s *Stealth) Encode(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, s.Version)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, s.Type)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, s.R)
	if err != nil {
		return err
	}
	err = s.TA.Encode(w)
	return err

}
