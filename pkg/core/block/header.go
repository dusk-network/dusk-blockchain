package block

import (
	"bytes"

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

func NewHeader() *Header {
	return &Header{
		Certificate: EmptyCertificate(),
	}
}

// SetHash will set this block header's hash by encoding all the relevant
// fields and then hashing the result.
func (b *Header) SetHash() error {
	buf := new(bytes.Buffer)
	if err := MarshalHashable(buf, b); err != nil {
		return err
	}

	h, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return err
	}

	b.Hash = h
	return nil
}

// Bytes returns the block header, encoded as a slice of bytes.
func (b *Header) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := MarshalHeader(buf, b)
	return buf.Bytes(), err
}

//Equals returns true if headers are equal
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
