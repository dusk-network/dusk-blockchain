package core

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"golang.org/x/crypto/ed25519"
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
// During reduction phase, it does not yet matter if we know whether we are voting
// on empty blocks or not, so it will always be set to false in the committeeVote
// function calls.
func (b *Blockchain) blockReduction(blockHash []byte) (bool, []byte, error) {
	// Step 1

	// Prepare empty block
	emptyBlock, err := payload.NewEmptyBlock(b.lastHeader)
	if err != nil {
		return false, nil, err
	}

	// If no candidate block was found, then we use the empty block
	if blockHash == nil {
		blockHash = emptyBlock.Header.Hash
	}

	// Vote on passed block
	if err := b.committeeVoteReduction(reductionThreshold1, 1, blockHash); err != nil {
		return false, nil, err
	}

	// Receive all other votes
	retHash, err := b.countVotesReduction(reductionThreshold1, reductionVoteThreshold1, 1, reductionTime1)
	if err != nil {
		return false, nil, err
	}

	// Step 2

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will vote on an empty block instead.
	if retHash == nil {
		if err := b.committeeVoteReduction(reductionThreshold2, 2, emptyBlock.Header.Hash); err != nil {
			return false, nil, err
		}
	} else {
		if err := b.committeeVoteReduction(reductionThreshold2, 2, retHash); err != nil {
			return false, nil, err
		}
	}

	retHash2, err := b.countVotesReduction(reductionThreshold2, reductionVoteThreshold2, 2, reductionTime2)
	if err != nil {
		return false, nil, err
	}

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will return an empty block instead.
	if retHash2 == nil {
		return true, emptyBlock.Header.Hash, nil
	}

	return false, retHash2, nil
}

func (b *Blockchain) committeeVoteReduction(threshold float64, step uint8, blockHash []byte) error {
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
		// Sign block hash with BLS
		sigBLS, err := bls.Sign(b.BLSSecretKey, blockHash)
		if err != nil {
			return err
		}

		// Create message to sign with ed25519
		var edMsg []byte // Add score when bls signatures are completed
		binary.LittleEndian.PutUint64(edMsg, b.round)
		edMsg = append(edMsg, byte(step))
		edMsg = append(edMsg, b.lastHeader.Hash...)
		// Add sigBLS once completed

		// Sign with ed25519
		sigEd := ed25519.Sign(*b.EdSecretKey, edMsg)

		// Create reduction message to gossip
		msg, err := payload.NewMsgReduction(score, blockHash, b.lastHeader.Hash, sigEd,
			b.EdPubKey, sigBLS, b.BLSPubKey, b.stakeWeight, b.round, step)
		if err != nil {
			return err
		}

		// Gossip msg
		msg.Command()
	}

	return nil
}

func (b *Blockchain) countVotesReduction(threshold float64, voteThreshold uint64, step uint8,
	timerAmount time.Duration) ([]byte, error) {
	counts := make(map[string]int)
	var voters []*ed25519.PublicKey
	timer := time.NewTimer(timerAmount)

	for {
	start:
		select {
		case <-timer.C:
			goto end
		case m := <-b.reductionChan:
			// Verify the message score and get back it's contents
			votes, hash, err := b.processMsgReduction(threshold, step, m)
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
				if voter == m.PubKeyEd {
					goto start
				}
			}

			// Log new information
			voters = append(voters, m.PubKeyEd)
			hashStr := hex.EncodeToString(hash)
			counts[hashStr] += votes

			// If a block exceeds the vote threshold, we will return it's hash
			// and end the loop.
			if counts[hashStr] > int(float64(voteThreshold)*threshold) {
				timer.Stop()
				return hash, nil
			}
		}
	}
end:

	return nil, nil
}

func (b *Blockchain) processMsgReduction(threshold float64, step uint8, msg *payload.MsgReduction) (int, []byte, error) {
	// Verify message
	if !verifySignaturesReduction(msg) {
		return 0, nil, nil
	}

	role := &role{
		part:  "committee",
		round: b.round,
		step:  step,
	}

	// Check if we're on the same chain
	if bytes.Compare(msg.PrevBlockHash, b.lastHeader.Hash) != 0 {
		// Either an old message or a malformed message
		return 0, nil, nil
	}

	// Make sure their score is valid, and calculate their amount of votes.
	votes, err := b.verifySortition(msg.Score, msg.PubKeyBLS, role, threshold, msg.Stake)
	if err != nil {
		return 0, nil, err
	}

	if votes == 0 {
		return 0, nil, nil
	}

	return votes, msg.BlockHash, nil
}

func verifySignaturesReduction(msg *payload.MsgReduction) bool {
	// Construct message
	var edMsg []byte // Add score when bls signatures are completed
	binary.LittleEndian.PutUint64(edMsg, msg.Round)
	edMsg = append(edMsg, byte(msg.Step))
	edMsg = append(edMsg, msg.PrevBlockHash...)
	// Add bls block sig once completed

	// Check ed25519
	if !ed25519.Verify(*msg.PubKeyEd, edMsg, msg.SigEd) {
		return false
	}

	// Check BLS
	if err := bls.Verify(msg.PubKeyBLS, msg.BlockHash, msg.SigBLS); err != nil {
		return false
	}

	// Passed all checks
	return true
}
