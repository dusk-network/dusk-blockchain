package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

type role struct {
	part  string
	round uint64
	step  uint8
}

// BlockReduction is the main function that runs during block reduction phase.
// Once the reduction phase is finished, the function will return a block hash,
// to then be used for the binary agreement phase.
func BlockReduction(ctx *Context, c chan *payload.MsgReduction) error {
	// Step 1
	ctx.step = 1

	// Prepare empty block
	emptyBlock, err := payload.NewEmptyBlock(ctx.GetLastHeader())
	if err != nil {
		return err
	}

	// If no candidate block was found, then we use the empty block
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.empty = true
	}

	// Vote on passed block
	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countVotesReduction(ctx, c); err != nil {
		return err
	}

	// Step 2
	ctx.step++
	ctx.AdjustVarsReduction()

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will vote on an empty block instead.
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.empty = true
	}

	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	if err := countVotesReduction(ctx, c); err != nil {
		return err
	}

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will return an empty block instead.
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.empty = true
	}

	return nil
}

func committeeVoteReduction(ctx *Context) error {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	if err := sortition(ctx, role); err != nil {
		return err
	}

	if ctx.votes > 0 {
		// Sign block hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.BlockHash)
		if err != nil {
			return err
		}

		// Create message to sign with ed25519
		var edMsg []byte
		copy(edMsg, ctx.Score)
		binary.LittleEndian.PutUint64(edMsg, ctx.Round)
		edMsg = append(edMsg, byte(ctx.step))
		edMsg = append(edMsg, ctx.GetLastHeader().Hash...)
		edMsg = append(edMsg, sigBLS...)

		// Sign with ed25519
		sigEd, err := ctx.EDSign(ctx.Keys.EdSecretKey, edMsg)
		if err != nil {
			return err
		}

		// Create reduction message to gossip
		blsPubBytes, err := ctx.Keys.BLSPubKey.MarshalBinary()
		if err != nil {
			return err
		}

		msg, err := payload.NewMsgReduction(ctx.Score, ctx.BlockHash, ctx.GetLastHeader().Hash, sigEd,
			[]byte(*ctx.Keys.EdPubKey), sigBLS, blsPubBytes, ctx.weight, ctx.Round, ctx.step)
		if err != nil {
			return err
		}

		if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
			return err
		}
	}

	return nil
}

func countVotesReduction(ctx *Context, c chan *payload.MsgReduction) error {
	counts := make(map[string]int)
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	counts[hex.EncodeToString(ctx.BlockHash)] += ctx.votes
	timer := time.NewTimer(ctx.Lambda)

end:
	for {
		select {
		case <-timer.C:
			break end
		case m := <-c:
			// Verify the message score and get back it's contents
			votes, hash, err := processMsgReduction(ctx, m)
			if err != nil {
				return err
			}

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 {
				break
			}

			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Compare(voter, m.PubKeyEd) == 0 {
					break
				}
			}

			// Log new information
			voters = append(voters, m.PubKeyEd)
			hashStr := hex.EncodeToString(hash)
			counts[hashStr] += votes

			// If a block exceeds the vote threshold, we will return it's hash
			// and end the loop.
			if counts[hashStr] > int(float64(ctx.VoteLimit)*ctx.Threshold) {
				timer.Stop()
				ctx.BlockHash = hash
				return nil
			}
		}
	}

	ctx.BlockHash = nil
	return nil
}

func processMsgReduction(ctx *Context, msg *payload.MsgReduction) (int, []byte, error) {
	// Verify message
	if !verifySignaturesReduction(ctx, msg) {
		return 0, nil, nil
	}

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	// Check if we're on the same chain
	if bytes.Compare(msg.PrevBlockHash, ctx.GetLastHeader().Hash) != 0 {
		// Either an old message or a malformed message
		return 0, nil, nil
	}

	// Make sure their score is valid, and calculate their amount of votes.
	votes, err := verifySortition(ctx, msg.Score, msg.PubKeyBLS, role, msg.Stake)
	if err != nil {
		return 0, nil, err
	}

	if votes == 0 {
		return 0, nil, nil
	}

	return votes, msg.BlockHash, nil
}

func verifySignaturesReduction(ctx *Context, msg *payload.MsgReduction) bool {
	// Construct message
	var edMsg []byte
	copy(edMsg, msg.Score)
	binary.LittleEndian.PutUint64(edMsg, msg.Round)
	edMsg = append(edMsg, byte(msg.Step))
	edMsg = append(edMsg, msg.PrevBlockHash...)
	edMsg = append(edMsg, msg.SigBLS...)

	// Check ed25519
	if !ctx.EDVerify(msg.PubKeyEd, edMsg, msg.SigEd) {
		return false
	}

	// Check BLS
	if err := ctx.BLSVerify(msg.PubKeyBLS, msg.BlockHash, msg.SigBLS); err != nil {
		return false
	}

	// Passed all checks
	return true
}
