package block

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-crypto/hash"
)

const (
	// HeaderHashSize size of a block header hash in bytes
	HeaderHashSize = 32
	// HeightSize size of a block height field in bytes
	HeightSize = 8
)

// Header defines a block header on a Dusk block.
type Header struct {
	Version   uint8  // Block version byte
	Height    uint64 // Block height
	Timestamp int64  // Block timestamp

	PrevBlockHash []byte // Hash of previous block (32 bytes)
	Seed          []byte // Marshaled BLS signature or hash of the previous block seed (32 bytes)
	TxRoot        []byte // Root hash of the merkle tree containing all txes (32 bytes)

	*Certificate        // Block certificate
	Hash         []byte // Hash of all previous fields
}

// NewHeader creates a new Block Header
func NewHeader() *Header {
	return &Header{
		Certificate: EmptyCertificate(),
	}
}

// CalculateHash will calculate and return this block header's hash by encoding all the relevant
// fields and then hashing the result.
func (b *Header) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalHashable(buf, b); err != nil {
		return nil, err
	}

	return hash.Sha3256(buf.Bytes())
}

// Equals returns true if headers are equal
func (b *Header) Equals(other *Header) bool {

	if other == nil {
		return false
	}

	if b.Version != other.Version {
		return false
	}

	if b.Timestamp != other.Timestamp {
		return false
	}

	if !bytes.Equal(b.PrevBlockHash, other.PrevBlockHash) {
		return false
	}

	if !bytes.Equal(b.Seed, other.Seed) {
		return false
	}

	if !bytes.Equal(b.TxRoot, other.TxRoot) {
		return false
	}

	if !b.Certificate.Equals(other.Certificate) {
		return false
	}

	if !bytes.Equal(b.Hash, other.Hash) {
		return false
	}

	return true
}

func marshalHashable(b *bytes.Buffer, h *Header) error {
	if err := binary.Write(b, binary.LittleEndian, h.Version); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, h.Height); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, uint64(h.Timestamp)); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.PrevBlockHash); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.Seed); err != nil {
		return err
	}

	return nil
}
