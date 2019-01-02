package transactions

import (
	"bytes"
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
)

// Stealth defines a stealth transaction.
type Stealth struct {
	Version  uint8  // 1 byte
	Type     TxType // 1 byte
	Inputs   []*Input
	TxPubKey []byte // 32 bytes
	Outputs  []*Output

	Hash []byte // 32 bytes
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
	s.TxPubKey = pk
}

// Clear all transaction fields
func (s *Stealth) Clear() {
	s.Inputs = nil
	s.TxPubKey = nil
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

	lIn := uint64(len(s.Inputs))
	if err := encoding.WriteVarInt(w, lIn); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.Write256(w, s.TxPubKey); err != nil {
		return err
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

	if err := encoding.Read256(r, &s.TxPubKey); err != nil {
		return err
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

	if err := encoding.Read256(r, &s.Hash); err != nil {
		return err
	}

	return nil
}

// CalculateHash implements merkletree.Payload
func (s Stealth) CalculateHash() ([]byte, error) {
	if s.Hash == nil {
		if err := s.SetHash(); err != nil {
			return nil, fmt.Errorf("CalculateHash(): error setting tx hash - %v", err)
		}
	}

	return s.Hash, nil
}

// Equals implements merkletree.Payload
func (s Stealth) Equals(other merkletree.Payload) (bool, error) {
	if s.Hash == nil {
		if err := s.SetHash(); err != nil {
			return false, fmt.Errorf("Equals(): error setting tx hash - %v", err)
		}
	}

	return bytes.Compare(s.Hash, other.(Stealth).Hash) == 0, nil
}
