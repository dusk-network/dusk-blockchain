// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package verifiers

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
)

// ErrPrevBlockHash previous block hash does not equal the previous hash in the current block.
var ErrPrevBlockHash = errors.New("previous block hash does not equal the previous hash in the current block")

// ErrInvalidBlockHash hashed set of block header fields is not equal to block.header.hash.
var ErrInvalidBlockHash = errors.New("invalid block hash")

// CheckBlockCertificate ensures that the block certificate is valid.
func CheckBlockCertificate(provisioners user.Provisioners, blk block.Block, seed []byte) error {
	// TODO: this should be set back to 1, once we fix this issue:
	// https://github.com/dusk-network/dusk-blockchain/issues/925
	if blk.Header.Height < 2 {
		return nil
	}

	// First, lets get the actual reduction steps
	// These would be the two steps preceding the one on the certificate
	stepOne := (blk.Header.Iteration-1)*3 + 2
	stepTwo := (blk.Header.Iteration-1)*3 + 2

	stepOneBatchedSig := blk.Header.Certificate.StepOneBatchedSig
	stepTwoBatchedSig := blk.Header.Certificate.StepTwoBatchedSig

	// Now, check the certificate's correctness for both reduction steps
	if err := checkBlockCertificateForStep(stepOneBatchedSig, blk.Header.Certificate.StepOneCommittee, blk.Header.Height, stepOne, provisioners, blk.Header.Hash, seed); err != nil {
		return err
	}

	return checkBlockCertificateForStep(stepTwoBatchedSig, blk.Header.Certificate.StepTwoCommittee, blk.Header.Height, stepTwo, provisioners, blk.Header.Hash, seed)
}

func checkBlockCertificateForStep(batchedSig []byte, bitSet uint64, round uint64, step uint8, provisioners user.Provisioners, blockHash, seed []byte) error {
	size := committeeSize(provisioners.SubsetSizeAt(round))
	committee := provisioners.CreateVotingCommittee(seed, round, step, size)
	subcommittee := committee.IntersectCluster(bitSet)

	apk, err := agreement.AggregatePks(&provisioners, subcommittee.Set)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, blockHash, apk, batchedSig)
}

func committeeSize(memberAmount int) int {
	if memberAmount > config.ConsensusMaxCommitteeSize {
		return config.ConsensusMaxCommitteeSize
	}

	return memberAmount
}

// CheckBlockHeader checks whether a block header is malformed.
// These are stateless and stateful checks.
// Returns nil, if all checks pass.
func CheckBlockHeader(prevBlock block.Block, blk block.Block) error {
	// Version
	if blk.Header.Version > 0 {
		return errors.New("unsupported block version")
	}

	if err := CheckHash(&blk); err != nil {
		return err
	}

	// blk.Headerheight = prevHeaderHeight +1
	if blk.Header.Height != prevBlock.Header.Height+1 {
		return errors.New("invalid block height")
	}

	// blk.Headerhash = prevHeaderHash
	if !bytes.Equal(blk.Header.PrevBlockHash, prevBlock.Header.Hash) {
		return ErrPrevBlockHash
	}

	// blk.Timestamp > prevTimestamp
	if blk.Header.Timestamp < prevBlock.Header.Timestamp {
		return errors.New("current timestamp is less than the previous timestamp")
	}

	if blk.Header.Height > 1 {
		if blk.Header.Timestamp > prevBlock.Header.Timestamp+config.MaxBlockTime {
			return errors.New("current timestamp is bigger than the prev timestamp + maxblocktime")
		}
	}

	if len(blk.Header.StateHash) != 32 {
		return errors.New("invalid state hash")
	}

	// Merkle tree check -- Check is here as the root is not calculated on decode
	root, err := blk.CalculateTxRoot()
	if err != nil {
		return errors.New("could not calculate the merkle tree root for this header")
	}

	if !bytes.Equal(root, blk.Header.TxRoot) {
		return errors.New("merkle root mismatch")
	}

	return nil
}

// CheckHash ensures that provided Header.Hash is valid.
func CheckHash(blk *block.Block) error {
	hash, err := blk.CalculateHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, blk.Header.Hash) {
		return ErrInvalidBlockHash
	}

	return nil
}
