// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package block

import (
	"bytes"
)

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	StepOneBatchedSig []byte `json:"step-one-batched-sig"` // Batched BLS signature of the block reduction phase (33 bytes)
	StepTwoBatchedSig []byte `json:"step-two-batched-sig"`
	StepOneCommittee  uint64 `json:"step-one-committee"` // Binary representation of the committee members who voted in favor of this block (8 bytes)
	StepTwoCommittee  uint64 `json:"step-two-committee"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (c *Certificate) Copy() *Certificate {
	cert := EmptyCertificate()

	if c.StepOneBatchedSig != nil {
		copy(cert.StepOneBatchedSig, c.StepOneBatchedSig)
	}

	if c.StepTwoBatchedSig != nil {
		copy(cert.StepTwoBatchedSig, c.StepTwoBatchedSig)
	}

	cert.StepOneCommittee = c.StepOneCommittee
	cert.StepTwoCommittee = c.StepTwoCommittee

	return cert
}

// EmptyCertificate returns an empty Certificate instance.
func EmptyCertificate() *Certificate {
	return &Certificate{
		StepOneBatchedSig: make([]byte, 33),
		StepTwoBatchedSig: make([]byte, 33),
		StepOneCommittee:  0,
		StepTwoCommittee:  0,
	}
}

// Equals returns true if both certificates are equal.
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

	if c.StepOneCommittee != other.StepOneCommittee {
		return false
	}

	if c.StepTwoCommittee != other.StepTwoCommittee {
		return false
	}

	return true
}
