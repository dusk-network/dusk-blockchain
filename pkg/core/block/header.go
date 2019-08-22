package block

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/hash"
)

const (
	// HeaderHashSize size of a block header hash in bytes
	HeaderHashSize = 32
	// HeightSize size of a block height field in bytes
	HeightSize = 8
)

func MarshalHashable(r io.Writer, h *Header) error {
	if err := encoding.WriteUint8(r, h.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, h.Height); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, uint64(h.Timestamp)); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, h.Seed); err != nil {
		return err
	}

	return nil
}

func MarshalHeader(r io.Writer, h *Header) error {
	if err := MarshalHashable(r, h); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.TxRoot); err != nil {
		return err
	}

	if err := MarshalCertificate(r, h.Certificate); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.Hash); err != nil {
		return err
	}

	return nil
}

func UnmarshalHeader(r io.Reader, h *Header) error {
	if err := encoding.ReadUint8(r, &h.Version); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &h.Height); err != nil {
		return err
	}

	var timestamp uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &timestamp); err != nil {
		return err
	}
	h.Timestamp = int64(timestamp)

	if err := encoding.Read256(r, &h.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &h.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, &h.TxRoot); err != nil {
		return err
	}

	if err := UnmarshalCertificate(r, h.Certificate); err != nil {
		return err
	}

	if err := encoding.Read256(r, &h.Hash); err != nil {
		return err
	}

	return nil
}

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
