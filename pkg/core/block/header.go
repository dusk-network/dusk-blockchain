package block

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Header defines a block header on a Dusk block.
type Header struct {
	Version   uint8  // Block version byte
	Timestamp int64  // Block timestamp
	Height    uint64 // Block height

	PrevBlock []byte // Hash of previous block (32 bytes)
	Seed      []byte // Marshaled BLS signature or hash of the previous block seed (32 bytes)
	TxRoot    []byte // Root hash of the merkle tree containing all txes (32 bytes)

	CertHash []byte // Hash of the block certificate (32 bytes)
	Hash     []byte // Hash of all previous fields
}

// SetHash will set this block header's hash by encoding all the relevant
// fields and then hashing the result.
func (b *Header) SetHash() error {
	buf := new(bytes.Buffer)
	if err := b.EncodeHashable(buf); err != nil {
		return err
	}

	h, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return err
	}

	b.Hash = h
	return nil
}

// EncodeHashable will encode all the fields needed from a Header to create
// a block hash. Result will be written to w.
func (b *Header) EncodeHashable(w io.Writer) error {
	if err := encoding.WriteUint8(w, b.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, b.Height); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, uint64(b.Timestamp)); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.PrevBlock); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, b.Seed); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.TxRoot); err != nil {
		return err
	}

	return nil
}

// Encode a Header struct and write to w.
func (b *Header) Encode(w io.Writer) error {
	if err := b.EncodeHashable(w); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.CertHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.Hash); err != nil {
		return err
	}

	return nil
}

// Decode a Header struct from r into b.
func (b *Header) Decode(r io.Reader) error {
	if err := encoding.ReadUint8(r, &b.Version); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &b.Height); err != nil {
		return err
	}

	var timestamp uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &timestamp); err != nil {
		return err
	}

	b.Timestamp = int64(timestamp)
	if err := encoding.Read256(r, &b.PrevBlock); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &b.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.TxRoot); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.CertHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.Hash); err != nil {
		return err
	}

	return nil
}

// Bytes returns the block header, encoded as a slice of bytes.
func (b *Header) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := b.Encode(buf)
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

	if !bytes.Equal(b.PrevBlock, other.PrevBlock) {
		return false
	}

	if !bytes.Equal(b.Seed, other.Seed) {
		return false
	}

	if !bytes.Equal(b.TxRoot, other.TxRoot) {
		return false
	}

	if !bytes.Equal(b.CertHash, other.CertHash) {
		return false
	}

	if !bytes.Equal(b.Hash, other.Hash) {
		return false
	}

	return true
}
