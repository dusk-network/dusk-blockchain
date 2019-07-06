package block

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	StepOneBatchedSig []byte // Batched BLS signature of the block reduction phase (33 bytes)
	StepTwoBatchedSig []byte
	Step              uint8  // Step the agreement terminated at (1 byte)
	StepOneCommittee  uint64 // Binary representation of the committee members who voted in favor of this block (8 bytes)
	StepTwoCommittee  uint64
}

func EmptyCertificate() *Certificate {
	return &Certificate{
		StepOneBatchedSig: make([]byte, 33),
		StepTwoBatchedSig: make([]byte, 33),
		Step:              0,
		StepOneCommittee:  0,
		StepTwoCommittee:  0,
	}
}

// Encode a Certificate struct and write to w.
func (c *Certificate) Encode(w io.Writer) error {
	if err := encoding.WriteBLS(w, c.StepOneBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, c.StepTwoBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, c.Step); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, c.StepOneCommittee); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, c.StepTwoCommittee); err != nil {
		return err
	}

	return nil
}

// Decode a Certificate struct from r into c.
func (c *Certificate) Decode(r io.Reader) error {
	if err := encoding.ReadBLS(r, &c.StepOneBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &c.StepTwoBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &c.Step); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &c.StepOneCommittee); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &c.StepTwoCommittee); err != nil {
		return err
	}

	return nil
}

// Equals returns true if both certificates are equal
func (c *Certificate) Equals(other *Certificate) bool {
	if other == nil {
		return false
	}

	if !bytes.Equal(c.StepOneBatchedSig, other.StepOneBatchedSig) {
		return false
	}

	if !bytes.Equal(c.StepTwoBatchedSig, other.StepTwoBatchedSig) {
		return false
	}

	if c.Step != other.Step {
		return false
	}

	if c.StepOneCommittee != other.StepOneCommittee {
		return false
	}

	if c.StepTwoCommittee != other.StepTwoCommittee {
		return false
	}

	return true
}
