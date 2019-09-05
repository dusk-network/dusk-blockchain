package block

import (
	"bytes"
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
