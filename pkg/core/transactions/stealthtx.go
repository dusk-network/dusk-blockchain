package transactions

import (
	"bytes"
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
	TypeInfo TypeInfo
	R        []byte // TxID (32 bytes)
}

// DecodeTransaction will decode a Stealth struct from r and return it.
func DecodeTransaction(r io.Reader) (*Stealth, error) {
	var version uint8
	if err := encoding.ReadUint8(r, &version); err != nil {
		return nil, err
	}

	var txType uint8
	if err := encoding.ReadUint8(r, &txType); err != nil {
		return nil, err
	}

	typeInfo, err := decodeTypeInfo(r, TxType(txType))
	if err != nil {
		return nil, err
	}

	var txID []byte
	if err := encoding.Read256(r, &txID); err != nil {
		return nil, err
	}

	return &Stealth{
		Version:  version,
		Type:     TxType(txType),
		TypeInfo: typeInfo,
		R:        txID,
	}, nil
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
