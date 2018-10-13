package transactions

import (
	"github.com/toghrulmaharramov/dusk-go/encoding"
	"io"
)

var (
	intSerializer = encoding.IntSerializer
	hashSerializer = encoding.HashSerializer
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
	if err := intSerializer.PutUint8(w, s.Version); err != nil {
		return err
	}

	if err := intSerializer.PutUint8(w, s.Type); err != nil {
		return err
	}

	if err := encoding.WriteHash(w, s.R); err != nil {
		return err
	}

	err := s.TA.Encode(w)
	return err
}

// GetEncodeSize will read through the stealth tx object to see how many bytes will have to be
// allocated in order to serialize all the data. This function can be used to pre-allocate the
// required amount of space to a bytes.Buffer before calling Encode, to reduce the amount of
// dynamic allocations being made while it runs.
func (s *Stealth) GetEncodeSize() uint64 {
	var size uint64

	size += 1 // Version
	size += 1 // Type
	size += 32 // R

	// TA
	// Inputs
	lenIn := uint64(len(s.TA.Inputs))
	size += uint64(encoding.VarIntSerializeSize(lenIn)) // Inputs length prefix
	size += 65 * lenIn // KeyImage, TxID, Index * amount of Inputs
	for _, input := range s.TA.Inputs {
		lenSig := uint64(len(input.Signature))
		size += uint64(encoding.VarIntSerializeSize(lenSig)) // Signature length prefix
		size += lenSig // Signature
	}

	lenOut := uint64(len(s.TA.Outputs))
	size += 32 // TxPubKey
	size += uint64(encoding.VarIntSerializeSize(lenOut)) // Outputs length prefix
	size += 40 * lenOut // Outputs

	return size
}