package block

import (
	"bytes"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	BRBatchedSig      []byte   // Batched BLS signature of the block reduction phase (33 bytes)
	BRStep            uint8    // Step the block reduction terminated at
	BRPubKeys         []byte   // Aggregated BLS public keys associated with the signature
	BRSortitionProofs [][]byte // Public key sortition proofs

	SRBatchedSig      []byte   // Batched BLS signature of the signature set reduction phase (33 bytes)
	SRStep            uint8    // Step the signature set reduction terminated at
	SRPubKeys         []byte   // Aggregated BLS public keys associated with the signature
	SRSortitionProofs [][]byte // Public key sortition proofs

	Hash []byte
}

// NewCertificate returns a Certificate struct with the provided signature.
func NewCertificate(brBatchedSig []byte, brStep uint8, brPubKeys []byte, brProofs [][]byte,
	srBatchedSig []byte, srStep uint8, srPubKeys []byte, srProofs [][]byte) *Certificate {
	return &Certificate{
		BRBatchedSig:      brBatchedSig,
		BRStep:            brStep,
		BRPubKeys:         brPubKeys,
		BRSortitionProofs: brProofs,

		SRBatchedSig:      srBatchedSig,
		SRStep:            srStep,
		SRPubKeys:         srPubKeys,
		SRSortitionProofs: srProofs,
	}
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

	if err := encoding.WriteUint8(w, c.BRStep); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, c.BRPubKeys); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(c.BRSortitionProofs))); err != nil {
		return err
	}

	for _, brProof := range c.BRSortitionProofs {
		if err := encoding.WriteVarBytes(w, brProof); err != nil {
			return err
		}
	}

	if err := encoding.WriteBLS(w, c.SRBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, c.SRStep); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, c.SRPubKeys); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(c.SRSortitionProofs))); err != nil {
		return err
	}

	for _, srProof := range c.SRSortitionProofs {
		if err := encoding.WriteVarBytes(w, srProof); err != nil {
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

	if err := encoding.ReadUint8(r, &c.BRStep); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &c.BRPubKeys); err != nil {
		return err
	}

	lBRProofs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.BRSortitionProofs = make([][]byte, lBRProofs)
	for i := uint64(0); i < lBRProofs; i++ {
		if err := encoding.ReadVarBytes(r, &c.BRSortitionProofs[i]); err != nil {
			return err
		}
	}

	if err := encoding.ReadBLS(r, &c.SRBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &c.SRStep); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &c.SRPubKeys); err != nil {
		return err
	}

	lSRProofs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.SRSortitionProofs = make([][]byte, lSRProofs)
	for i := uint64(0); i < lSRProofs; i++ {
		if err := encoding.ReadVarBytes(r, &c.SRSortitionProofs[i]); err != nil {
			return err
		}
	}

	if err := encoding.Read256(r, &c.Hash); err != nil {
		return err
	}

	return nil
}
