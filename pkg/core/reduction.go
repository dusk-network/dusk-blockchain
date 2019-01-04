package core

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
)

var (
	reductionThreshold1            = 0.1
	reductionThreshold2            = 0.1
	reductionTime1                 = 20 * time.Second
	reductionTime2                 = 20 * time.Second
	reductionVoteThreshold1 uint64 = 10
	reductionVoteThreshold2 uint64 = 10
)

type role struct {
	part  string
	round uint64
	step  uint8
}

// BlockReduction is the main function that runs during block reduction phase.
// Once the reduction phase is finished, the function will return a block hash,
// to then be used for the binary agreement phase.
func (b *Blockchain) BlockReduction(blockHash []byte) ([]byte, error) {
	// Step 1

	// Prepare empty block
	emptyBlock, err := payload.NewEmptyBlock(b.lastHeader)
	if err != nil {
		return nil, err
	}

	// If no candidate block was found, then we use the empty block
	if blockHash == nil {
		blockHash = emptyBlock.Header.Hash
	}

	// Vote on passed block
	if err := b.committeeVote(reductionThreshold1, 1, blockHash); err != nil {
		return nil, err
	}

	// Receive all other votes
	retHash, err := b.countVotes(reductionThreshold1, 1, reductionTime1)
	if err != nil {
		return nil, err
	}

	// Step 2

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will vote on an empty block instead.
	if retHash == nil {
		if err := b.committeeVote(reductionThreshold2, 2, emptyBlock.Header.Hash); err != nil {
			return nil, err
		}
	} else {
		if err := b.committeeVote(reductionThreshold2, 2, retHash); err != nil {
			return nil, err
		}
	}

	retHash2, err := b.countVotes(reductionThreshold2, 2, reductionTime2)
	if err != nil {
		return nil, err
	}

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will return an empty block instead.
	if retHash2 == nil {
		return emptyBlock.Header.Hash, nil
	}

	return retHash2, nil
}

func (b *Blockchain) committeeVote(threshold float64, step uint8, blockHash []byte) error {
	role := &role{
		part:  "committee",
		round: b.round,
		step:  step,
	}

	score, j, err := b.sortition(role, threshold)
	if err != nil {
		return err
	}

	if j > 0 {
		prevBlockHash, err := b.GetLatestHeaderHash()
		if err != nil {
			return err
		}

		// TODO: Make Ed25519 sig

		// Sign block hash with BLS
		sigBLS, err := bls.Sign(b.BLSSecretKey, blockHash)
		if err != nil {
			return err
		}

		// Create message to gossip
		msg, err := payload.NewMsgReduction(score, blockHash, prevBlockHash, []byte{}, []byte{},
			sigBLS, b.BLSPubKey, b.stakeWeight, b.round, step)
		if err != nil {
			return err
		}

		// Gossip msg
		msg.Command() // placeholder for error
	}

	return nil
}

func (b *Blockchain) countVotes(threshold float64, step uint8, timerAmount time.Duration) ([]byte, error) {
	counts := make(map[string]int)
	var voters [][]byte
	timer := time.NewTimer(timerAmount)

	for {
	start:
		select {
		case <-timer.C:
			goto end
		case m := <-b.reductionChan:
			// Verify the message score and get back it's contents
			votes, pk, hash, err := b.processMsgReduction(threshold, step, m)
			if err != nil {
				return nil, err
			}

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 {
				goto start
			}

			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Compare(voter, pk) == 0 {
					goto start
				}
			}

			// Log new information
			voters = append(voters, pk)
			hashStr := hex.EncodeToString(hash)
			counts[hashStr] += votes

			// If a block exceeds the vote threshold, we will return it's hash
			// and end the loop.
			if counts[hashStr] > int(float64(reductionVoteThreshold1)*threshold) {
				timer.Stop()
				return hash, nil
			}
		}
	}
end:

	return nil, nil
}

func (b *Blockchain) processMsgReduction(threshold float64, step uint8, msg *payload.MsgReduction) (int, []byte, []byte, error) {
	// Verify signature and message
	// Add once Ed25519 code is added
	// if !edwards25519.Verify(msg.PubKey, msg.SigEd) {
	//	return 0, nil, nil, errors.New("mismatch between signature and public key")
	// }

	role := &role{
		part:  "committee",
		round: b.round,
		step:  step,
	}

	// Check if we're on the same chain
	lastHash, err := b.GetLatestHeaderHash()
	if err != nil {
		return 0, nil, nil, err
	}

	if bytes.Compare(msg.PrevBlockHash, lastHash) != 0 {
		// Either an old message or a malformed message
		return 0, nil, nil, nil
	}

	// Make sure their score is valid, and calculate their amount of votes.
	votes, err := b.verifySortition(msg.Score, msg.PubKeyBLS, role, threshold, msg.Stake)
	if err != nil {
		return 0, nil, nil, err
	}

	if votes == 0 {
		return 0, nil, nil, nil
	}

	return votes, msg.PubKeyEd, msg.BlockHash, nil
}
