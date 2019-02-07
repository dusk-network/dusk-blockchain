package transactions

import (
	"bytes"
	"encoding/hex"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// TypeInfo is an interface for type-specific transaction information.
type TypeInfo interface {
	Encode(w io.Writer) error
	Decode(r io.Reader) error
	Type() TxType
}

// Stealth defines a stealth transaction.
type Stealth struct {
	Version  uint8  // 1 byte
	Type     TxType // 1 byte
	TypeInfo TypeInfo
	R        []byte // TxID (32 bytes)
}

// NewTX will return a transaction with the specified type and type info.
func NewTX(t TxType, info TypeInfo) *Stealth {
	return &Stealth{
		Version:  0x00,
		Type:     t,
		TypeInfo: info,
	}
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

	s.R = h
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

	if s.TypeInfo != nil {
		if err := s.TypeInfo.Encode(w); err != nil {
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

	if err := encoding.Write256(w, s.R); err != nil {
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

	switch s.Type {
	case CoinbaseType:
		typeInfo := &Coinbase{}
		if err := typeInfo.Decode(r); err != nil {
			return err
		}

		s.TypeInfo = typeInfo
	case BidType:
		typeInfo := &Bid{}
		if err := typeInfo.Decode(r); err != nil {
			return err
		}

		s.TypeInfo = typeInfo
	case StakeType:
		typeInfo := &Stake{}
		if err := typeInfo.Decode(r); err != nil {
			return err
		}

		s.TypeInfo = typeInfo
	case StandardType:
		typeInfo := &Standard{}
		if err := typeInfo.Decode(r); err != nil {
			return err
		}

		s.TypeInfo = typeInfo
	case TimelockType:
		typeInfo := &Timelock{}
		if err := typeInfo.Decode(r); err != nil {
			return err
		}

		s.TypeInfo = typeInfo
	case ContractType:
		typeInfo := &Contract{}
		if err := typeInfo.Decode(r); err != nil {
			return err
		}

		s.TypeInfo = typeInfo
	}

	if err := encoding.Read256(r, &s.R); err != nil {
		return err
	}

	return nil
}

// Hex returns the tx hash as a hexadecimal string
func (s *Stealth) Hex() string {
	return hex.EncodeToString(s.R)
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
