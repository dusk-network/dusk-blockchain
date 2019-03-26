package block

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	BRBatchedSig []byte   // Batched BLS signature of the block reduction phase (33 bytes)
	BRStep       uint32   // Step the block reduction terminated at
	BRPubKeys    [][]byte // BLS public keys associated with the signature

	SRBatchedSig []byte   // Batched BLS signature of the signature set reduction phase (33 bytes)
	SRStep       uint32   // Step the signature set reduction terminated at
	SRPubKeys    [][]byte // BLS public keys associated with the signature

	Hash []byte
}

// SetHash will set the Certificate hash.
func (c *Certificate) SetHash() error {
	buf := new(bytes.Buffer)
	if err := c.EncodeHashable(buf); err != nil {
		return err
	}

	h, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return err
	}

	c.Hash = h
	return nil
}

// EncodeHashable will encode all fields needed from the CertificateStruct to create
// a certificate hash. Result will be written to w.
func (c *Certificate) EncodeHashable(w io.Writer) error {
	if err := encoding.WriteBLS(w, c.BRBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, c.BRStep); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(c.BRPubKeys))); err != nil {
		return err
	}

	for _, brpk := range c.BRPubKeys {
		if err := encoding.WriteVarBytes(w, brpk); err != nil {
			return err
		}
	}

	if err := encoding.WriteBLS(w, c.SRBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, c.SRStep); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(c.SRPubKeys))); err != nil {
		return err
	}

	for _, srpk := range c.SRPubKeys {
		if err := encoding.WriteVarBytes(w, srpk); err != nil {
			return err
		}
	}

	return nil
}

// Encode a Certificate struct and write to w.
func (c *Certificate) Encode(w io.Writer) error {
	if err := c.EncodeHashable(w); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.Hash); err != nil {
		return err
	}

	return nil
}

// Decode a Certificate struct from r into c.
func (c *Certificate) Decode(r io.Reader) error {
	if err := encoding.ReadBLS(r, &c.BRBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint32(r, binary.LittleEndian, &c.BRStep); err != nil {
		return err
	}

	lBRPubKeys, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.BRPubKeys = make([][]byte, lBRPubKeys)
	for i := uint64(0); i < lBRPubKeys; i++ {
		if err := encoding.ReadVarBytes(r, &c.BRPubKeys[i]); err != nil {
			return err
		}
	}

	if err := encoding.ReadBLS(r, &c.SRBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint32(r, binary.LittleEndian, &c.SRStep); err != nil {
		return err
	}

	lSRPubKeys, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.SRPubKeys = make([][]byte, lSRPubKeys)
	for i := uint64(0); i < lSRPubKeys; i++ {
		if err := encoding.ReadVarBytes(r, &c.SRPubKeys[i]); err != nil {
			return err
		}
	}

	if err := encoding.Read256(r, &c.Hash); err != nil {
		return err
	}

	return nil
}

// Equals returns true if both certificates are equal
func (c *Certificate) Equals(other *Certificate) bool {

	if other == nil {
		return false
	}

	if !bytes.Equal(c.BRBatchedSig, other.BRBatchedSig) {
		return false
	}

	if c.BRStep != other.BRStep {
		return false
	}

	if len(c.BRPubKeys) != len(other.BRPubKeys) {
		return false
	}

	if len(c.SRPubKeys) != len(other.SRPubKeys) {
		return false
	}

	for i := range c.BRPubKeys {

		brPubKey := c.BRPubKeys[i]
		otherBrPubKey := other.BRPubKeys[i]

		if !bytes.Equal(brPubKey, otherBrPubKey) {
			return false
		}
	}

	for i := range c.SRPubKeys {

		srPubKey := c.SRPubKeys[i]
		otherSrPubKey := other.SRPubKeys[i]

		if !bytes.Equal(srPubKey, otherSrPubKey) {
			return false
		}
	}

	return true

}
