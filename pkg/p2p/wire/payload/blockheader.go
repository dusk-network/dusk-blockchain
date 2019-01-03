package payload

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// BlockHeader defines a block header on a Dusk block.
type BlockHeader struct {
	Height    uint64 // Block height
	Timestamp int64  // Block timestamp

	PrevBlock []byte // Hash of previous block (32 bytes)
	Seed      []byte // BLS signature of the previous block seed (32 bytes)
	TxRoot    []byte // Root hash of the merkle tree containing all txes (32 bytes)

	Hash      []byte // Hash of all previous fields
	CertImage []byte // Hash of the block certificate (32 bytes)
}

// SetHash will set this block header's hash by encoding all the relevant
// fields and then hashing the result.
func (b *BlockHeader) SetHash() error {
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

// EncodeHashable will encode all the fields needed from a BlockHeader to create
// a block hash. Result will be written to w.
func (b *BlockHeader) EncodeHashable(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, b.Height); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, uint64(b.Timestamp)); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.PrevBlock); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.Seed); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.TxRoot); err != nil {
		return err
	}

	return nil
}

// Encode a BlockHeader struct and write to w.
func (b *BlockHeader) Encode(w io.Writer) error {
	if err := b.EncodeHashable(w); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.Hash); err != nil {
		return err
	}

	if err := encoding.Write256(w, b.CertImage); err != nil {
		return err
	}

	return nil
}

// Decode a Blockheader struct from r into b.
func (b *BlockHeader) Decode(r io.Reader) error {
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

	if err := encoding.Read256(r, &b.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.TxRoot); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.Hash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &b.CertImage); err != nil {
		return err
	}

	return nil
}

// Bytes returns the encoded bytes of a block header
func (b *BlockHeader) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := b.Encode(buf)
	return buf.Bytes(), err
}
