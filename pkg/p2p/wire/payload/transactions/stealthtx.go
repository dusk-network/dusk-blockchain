package transactions

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Stealth defines a stealth transaction.
type Stealth struct {
	Version  uint8  // 1 byte
	Type     TxType // 1 byte
	R        []byte // 32 bytes
	Inputs   []*Input
	Outputs  []*Output
	LockTime uint64 // 8 bytes
	Hash     []byte // 32 bytes
}

// NewTX will return a standard transaction
func NewTX() *Stealth {
	return &Stealth{
		Version: 0x00,
		Type:    StandardType,
	}
}

// AddInput will add an input to the transaction's TypeAttributes
func (s *Stealth) AddInput(input *Input) {
	s.Inputs = append(s.Inputs, input)
}

// AddOutput will add an output to the transaction's TypeAttributes
func (s *Stealth) AddOutput(output *Output) {
	s.Outputs = append(s.Outputs, output)
}

// AddTxPubKey will add a transaction public key to the transaction object
func (s *Stealth) AddTxPubKey(pk []byte) {
	// This should be adjusted in the future to work as described in
	// the documentation - this is merely a placeholder
	s.R = pk
}

// Clear all transaction fields
func (s *Stealth) Clear() {
	s.Inputs = nil
	s.R = nil
	s.Outputs = nil
	s.Hash = nil
}

// SetHash will set the transaction hash
func (s *Stealth) SetHash() error {
	buf := new(bytes.Buffer)
	if err := s.EncodeHashable(buf); err != nil {
		return err
	}

	h, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return err
	}

	s.Hash = h
	return nil
}

// EncodeHashable will encode all fields needed to create a
// tx hash.
func (s *Stealth) EncodeHashable(w io.Writer) error {
	if err := encoding.WriteUint8(w, s.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, uint8(s.Type)); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.R); err != nil {
		return err
	}

	lIn := uint64(len(s.Inputs))
	if err := encoding.WriteVarInt(w, lIn); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}

	lOut := uint64(len(s.Outputs))
	if err := encoding.WriteVarInt(w, lOut); err != nil {
		return err
	}

	for _, output := range s.Outputs {
		if err := output.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, s.LockTime); err != nil {
		return err
	}

	return nil
}

// Encode a Stealth object and write to w.
func (s *Stealth) Encode(w io.Writer) error {
	if err := s.EncodeHashable(w); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.Hash); err != nil {
		return err
	}

	return nil
}

// Decode a Stealth object from r into s.
func (s *Stealth) Decode(r io.Reader) error {
	if err := encoding.ReadUint8(r, &s.Version); err != nil {
		return err
	}

	var t uint8
	if err := encoding.ReadUint8(r, &t); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.R); err != nil {
		return err
	}

	s.Type = TxType(t)
	lIn, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.Inputs = make([]*Input, lIn)
	for i := uint64(0); i < lIn; i++ {
		s.Inputs[i] = &Input{}
		if err := s.Inputs[i].Decode(r); err != nil {
			return err
		}
	}

	lOut, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.Outputs = make([]*Output, lOut)
	for i := uint64(0); i < lOut; i++ {
		s.Outputs[i] = &Output{}
		if err := s.Outputs[i].Decode(r); err != nil {
			return err
		}
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &s.LockTime); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.Hash); err != nil {
		return err
	}

	return nil
}

// Hex returns the tx hash as a hexadecimal string
func (s *Stealth) Hex() string {
	return hex.EncodeToString(s.Hash)
}

// CalculateHash implements merkletree.Payload
func (s Stealth) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := s.EncodeHashable(buf); err != nil {
		return nil, err
	}

	return hash.Sha3256(buf.Bytes())
}

// Equals implements merkletree.Payload
func (s Stealth) Equals(other merkletree.Payload) (bool, error) {
	h, err := s.CalculateHash()
	if err != nil {
		return false, err
	}

	otherHash, err := other.(Stealth).CalculateHash()
	if err != nil {
		return false, err
	}

	return bytes.Compare(h, otherHash) == 0, nil
}
