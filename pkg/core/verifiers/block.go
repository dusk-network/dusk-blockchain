package verifiers

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
)

// CheckBlock will verify whether a block is valid according to the rules of the consensus
// returns nil if a block is valid
func CheckBlock(db database.DB, prevBlock block.Block, blk block.Block) error {
	// 1. Check that we have not seen this block before
	err := db.View(func(t database.Transaction) error {
		_, err := t.FetchBlockExists(blk.Header.Hash)
		return err
	})

	if err != database.ErrBlockNotFound {
		if err == nil {
			err = errors.New("block already exists")
		}
		return err
	}

	if err := CheckBlockHeader(prevBlock, blk); err != nil {
		return err
	}

	if err := CheckMultiCoinbases(blk.Txs); err != nil {
		return err
	}

	for i, merklePayload := range blk.Txs {
		tx, ok := merklePayload.(transactions.Transaction)
		if !ok {
			return errors.New("tx does not implement the transaction interface")
		}
		if err := CheckTx(db, uint64(i), uint64(blk.Header.Timestamp), tx); err != nil {
			return err
		}
	}
	return nil
}

// CheckBlockCertificate ensures that the block certificate is valid.
func CheckBlockCertificate(committee committee.Foldable, blk block.Block) error {
	if blk.Header.Height < 2 {
		return nil
	}

	// First, lets get the actual reduction steps
	// This would be the certificate step * 2 - 1, and certificate step * 2
	stepOne := (blk.Header.Certificate.Step * 2) - 1
	stepTwo := blk.Header.Certificate.Step * 2

	// Reconstruct signatures
	stepOneBatchedSig, err := bls.UnmarshalSignature(blk.Header.Certificate.StepOneBatchedSig)
	if err != nil {
		return err
	}

	stepTwoBatchedSig, err := bls.UnmarshalSignature(blk.Header.Certificate.StepTwoBatchedSig)
	if err != nil {
		return err
	}

	// Now, check the certificate's correctness for both reduction steps
	if err := checkBlockCertificateForStep(stepOneBatchedSig, blk.Header.Certificate.StepOneCommittee, blk.Header.Height, stepOne, committee, blk.Header.Hash); err != nil {
		return err
	}

	return checkBlockCertificateForStep(stepTwoBatchedSig, blk.Header.Certificate.StepTwoCommittee, blk.Header.Height, stepTwo, committee, blk.Header.Hash)
}

func checkBlockCertificateForStep(batchedSig *bls.Signature, bitSet uint64, round uint64, step uint8, committee committee.Foldable, blockHash []byte) error {
	subcommittee := committee.Unpack(bitSet, round, step)
	apk, err := agreement.ReconstructApk(subcommittee)
	if err != nil {
		return err
	}

	return agreement.VerifySignatures(round, step, blockHash, apk, batchedSig)
}

// CheckBlockHeader checks whether a block header is malformed,
// These are stateless and stateful checks
// returns nil, if all checks pass
func CheckBlockHeader(prevBlock block.Block, blk block.Block) error {
	// Version
	if blk.Header.Version > 0 {
		return errors.New("unsupported block version")
	}

	// blk.Headerhash = prevHeaderHash
	if !bytes.Equal(blk.Header.PrevBlockHash, prevBlock.Header.Hash) {
		return errors.New("Previous block hash does not equal the previous hash in the current block")
	}

	// blk.Headerheight = prevHeaderHeight +1
	if blk.Header.Height != prevBlock.Header.Height+1 {
		return errors.New("current block height is not one plus the previous block height")
	}

	// blk.Timestamp > prevTimestamp
	if blk.Header.Timestamp <= prevBlock.Header.Timestamp {
		return errors.New("current timestamp is less than the previous timestamp")
	}

	// Merkle tree check -- Check is here as the root is not calculated on decode
	tR := blk.Header.TxRoot
	if err := blk.SetRoot(); err != nil {
		return errors.New("could not calculate the merkle tree root for this header")
	}

	if !bytes.Equal(tR, blk.Header.TxRoot) {
		return errors.New("merkle root mismatch")
	}

	return nil
}

// CheckMultiCoinbases returns an error if there is more than one coinbase transaction
//  in the list or if there are none
func CheckMultiCoinbases(txs []transactions.Transaction) error {
	var seen bool
	for _, tx := range txs {
		if tx.Type() != transactions.CoinbaseType {
			continue
		}
		if seen {
			return errors.New("multiple coinbase transactions present")
		}
		seen = true
	}

	if !seen {
		return errors.New("no coinbase transactions in the list")
	}
	return nil
}
