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
	Seed      []byte // Marshaled BLS signature of the previous block seed (33 bytes)
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

// DecodeBlockHeader will decode a Header struct from r and return it.
func DecodeBlockHeader(r io.Reader) (*Header, error) {
	var version uint8
	if err := encoding.ReadUint8(r, &version); err != nil {
		return nil, err
	}

	var timestamp uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &timestamp); err != nil {
		return nil, err
	}

	var height uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &height); err != nil {
		return nil, err
	}

	var prevBlock []byte
	if err := encoding.Read256(r, &prevBlock); err != nil {
		return nil, err
	}

	var seed []byte
	if err := encoding.ReadBLS(r, &seed); err != nil {
		return nil, err
	}

	var txRoot []byte
	if err := encoding.Read256(r, &txRoot); err != nil {
		return nil, err
	}

	var certHash []byte
	if err := encoding.Read256(r, &certHash); err != nil {
		return nil, err
	}

	var hash []byte
	if err := encoding.Read256(r, &hash); err != nil {
		return nil, err
	}

	return &Header{
		Version:   version,
		Timestamp: int64(timestamp),
		Height:    height,
		PrevBlock: prevBlock,
		Seed:      seed,
		TxRoot:    txRoot,
		CertHash:  certHash,
		Hash:      hash,
	}, nil
}

// Bytes returns the block header, encoded as a slice of bytes.
func (b *Header) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := b.Encode(buf)
	return buf.Bytes(), err
}
