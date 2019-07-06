package block

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	BatchedSig []byte // Batched BLS signature of the block reduction phase (33 bytes)
	Step       uint8  // Step the block reduction terminated at (1 byte)
	Committee  uint64 // Binary representation of the committee members who voted in favor of this block (8 bytes)
}

// Encode a Certificate struct and write to w.
func (c *Certificate) Encode(w io.Writer) error {
	if err := encoding.WriteBLS(w, c.BatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, c.Step); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, c.Committee); err != nil {
		return err
	}

	return nil
}

// Decode a Certificate struct from r into c.
func (c *Certificate) Decode(r io.Reader) error {
	if err := encoding.ReadBLS(r, &c.BatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &c.Step); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &c.Committee); err != nil {
		return err
	}

	return nil
}

// Equals returns true if both certificates are equal
func (c *Certificate) Equals(other *Certificate) bool {
	if other == nil {
		return false
	}

	if !bytes.Equal(c.BatchedSig, other.BatchedSig) {
		return false
	}

	if c.Step != other.Step {
		return false
	}

	if c.Committee != other.Committee {
		return false
	}

	return true
}
