package core

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"golang.org/x/crypto/ed25519"
)

var (
	maxSteps uint8 = 100
)

func (b *Blockchain) binaryAgreement(blockHash []byte, empty bool) (bool, []byte, error) {
	// Prepare empty block
	emptyBlock, err := payload.NewEmptyBlock(b.lastHeader)
	if err != nil {
		return false, nil, err
	}

	for step := uint8(1); step < maxSteps; step++ {
		var retHash []byte
		var msgs []*payload.MsgBinary
		var err error
		if err := b.committeeVoteBinary(reductionThreshold1, step, blockHash, empty); err != nil {
			return false, nil, err
		}

		empty, retHash, _, err = b.countVotesBinary(reductionThreshold1, reductionVoteThreshold1, step, reductionTime1)
		if err != nil {
			return false, nil, err
		}

		// if step > 1 {

		// }

		if retHash == nil {
			retHash = blockHash
		} else {
			if !empty {
				if step == 1 {
					if err := b.committeeVoteBinary(reductionThreshold1, maxSteps, retHash, empty); err != nil {
						return false, nil, err
					}
				}

				return empty, retHash, nil
			}
		}

		step++
		if err := b.committeeVoteBinary(reductionThreshold1, step, retHash, empty); err != nil {
			return false, nil, err
		}

		empty, retHash, _, err = b.countVotesBinary(reductionThreshold1, reductionVoteThreshold1, step, reductionTime1)
		if retHash == nil {
			retHash = emptyBlock.Header.Hash
			empty = true
		} else {
			if empty {
				return empty, retHash, nil
			}
		}

		step++
		if err := b.committeeVoteBinary(reductionThreshold1, step, retHash, empty); err != nil {
			return false, nil, err
		}

		empty, retHash, msgs, err = b.countVotesBinary(reductionThreshold1, reductionVoteThreshold1, step, reductionTime1)
		if retHash == nil {
			result, err := b.commonCoin(msgs, step)
			if err != nil {
				return false, nil, err
			}

			if result == 0 {
				retHash = blockHash
			} else {
				retHash = emptyBlock.Header.Hash
			}
		}
	}

	return true, emptyBlock.Header.Hash, nil
}

func (b *Blockchain) commonCoin(allMsgs []*payload.MsgBinary, step uint8) (uint64, error) {
	var lenHash, _ = new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
	for i, vote := range allMsgs {
		votes, h, err := b.processMsgBinary(reductionThreshold1, step, vote)
		if err != nil {
			return 0, err
		}

		for j := 1; j < votes; j++ {
			binary.LittleEndian.PutUint32(h, uint32(i))
			result, err := hash.Sha3256(h)
			if err != nil {
				return 0, err
			}

			resultInt := new(big.Int).SetBytes(result)
			if resultInt.Cmp(lenHash) == -1 {
				lenHash = resultInt
			}
		}
	}

	lenHash.Mod(lenHash, big.NewInt(2))
	return lenHash.Uint64(), nil
}

func (b *Blockchain) committeeVoteBinary(threshold float64, step uint8, blockHash []byte, empty bool) error {
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
		msg, err := payload.NewMsgBinary(score, empty, blockHash, b.lastHeader.Hash, sigEd,
			b.EdPubKey, sigBLS, b.BLSPubKey, b.stakeWeight, b.round, step)
		if err != nil {
			return err
		}

		// Gossip msg
		msg.Command()
	}

	return nil
}

func (b *Blockchain) countVotesBinary(threshold float64, voteThreshold uint64, step uint8,
	timerAmount time.Duration) (bool, []byte, []*payload.MsgBinary, error) {
	counts := make(map[string]int)
	var voters []*ed25519.PublicKey
	var allMsgs []*payload.MsgBinary
	timer := time.NewTimer(timerAmount)

	for {
	start:
		select {
		case <-timer.C:
			goto end
		case m := <-b.binaryChan:
			// Verify the message score and get back it's contents
			votes, hash, err := b.processMsgBinary(threshold, step, m)
			if err != nil {
				return false, nil, nil, err
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

			// Save vote for common coin
			allMsgs = append(allMsgs, m)

			// If a block exceeds the vote threshold, we will return it's hash
			// and end the loop.
			if counts[hashStr] > int(float64(voteThreshold)*threshold) {
				timer.Stop()
				return m.Empty, hash, allMsgs, nil
			}
		}
	}
end:

	return false, nil, allMsgs, nil
}

func (b *Blockchain) processMsgBinary(threshold float64, step uint8, msg *payload.MsgBinary) (int, []byte, error) {
	// Verify message
	if !verifySignaturesBinary(msg) {
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

func verifySignaturesBinary(msg *payload.MsgBinary) bool {
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
