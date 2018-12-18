package transactions

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
)

// Stealth defines a stealth transaction.
type Stealth struct {
	Version uint8          // 1 byte
	Type    uint8          // 1 byte
	R       []byte         // 32 bytes
	TA      TypeAttributes // (m * 2565) + 32 + (n * 40) m = # inputs, n = # of outputs
}

// Encode will serialize a stealthtx in byte format to w.
func (s *Stealth) Encode(w io.Writer) error {
	// Version
	if err := encoding.WriteUint8(w, s.Version); err != nil {
		return err
	}

	// Type
	if err := encoding.WriteUint8(w, s.Type); err != nil {
		return err
	}

	// R
	if err := encoding.Write256(w, s.R); err != nil {
		return err
	}

	// TA
	err := s.TA.Encode(w)
	return err
}

// Decode will deserialize a stealthtx from r and populate the Stealth object it was passed.
func (s *Stealth) Decode(r io.Reader) error {
	// Version
	if err := encoding.ReadUint8(r, &s.Version); err != nil {
		return err
	}

	// Type
	if err := encoding.ReadUint8(r, &s.Type); err != nil {
		return err
	}

	// R
	if err := encoding.Read256(r, &s.R); err != nil {
		return err
	}

	// TA
	return s.TA.Decode(r)

}

// GetEncodeSize will read through the stealth tx object to see how many bytes will have to be
// allocated in order to serialize all the data. This function can be used to pre-allocate the
// required amount of space to a bytes.Buffer before calling Encode, to reduce the amount of
// dynamic allocations being made while it runs.
func (s *Stealth) GetEncodeSize() uint64 {
	var size uint64

	size++     // Version
	size++     // Type
	size += 32 // R

	// TA
	// Inputs
	lenIn := uint64(len(s.TA.Inputs))
	size += uint64(encoding.VarIntEncodeSize(lenIn)) // Inputs length prefix
	size += 65 * lenIn                               // KeyImage, TxID, Index * amount of Inputs
	for _, input := range s.TA.Inputs {
		lenSig := uint64(len(input.Signature))
		size += uint64(encoding.VarIntEncodeSize(lenSig)) // Signature length prefix
		size += lenSig                                    // Signature
	}

	// Outputs
	lenOut := uint64(len(s.TA.Outputs))
	size += 32                                        // TxPubKey
	size += uint64(encoding.VarIntEncodeSize(lenOut)) // Outputs length prefix
	size += 40 * lenOut                               // Outputs

	return size
}
