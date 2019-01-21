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
// Once the reduction phase is finished, the context object holds a block hash
// that will then be voted on in the binary agreement phase.
func BlockReduction(ctx *Context, c chan *payload.MsgReduction) error {
	// Step 1
	ctx.step = 1

	// Prepare empty block
	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		return err
	}

	// If no candidate block was found, then we use the empty block
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.Empty = true
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

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will vote on an empty block instead.
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.Empty = true
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
		ctx.Empty = true
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
		edMsg = append(edMsg, ctx.Score...)
		binary.LittleEndian.PutUint64(edMsg, ctx.Round)
		edMsg = append(edMsg, byte(ctx.step))
		edMsg = append(edMsg, ctx.LastHeader.Hash...)
		edMsg = append(edMsg, sigBLS...)

		// Sign with ed25519
		sigEd := ctx.EDSign(ctx.Keys.EdSecretKey, edMsg)

		// Create reduction message to gossip
		blsPubBytes, err := ctx.Keys.BLSPubKey.MarshalBinary()
		if err != nil {
			return err
		}

		msg, err := payload.NewMsgReduction(ctx.Score, ctx.BlockHash, ctx.LastHeader.Hash, sigEd,
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
	counts := make(map[string]uint64)
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	counts[hex.EncodeToString(ctx.BlockHash)] += ctx.votes
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			ctx.BlockHash = nil
			return nil
		case m := <-c:
			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKeyEd) {
					break out
				}
			}

			// Verify the message score and get back it's contents
			votes, err := processMsgReduction(ctx, m)
			if err != nil {
				return err
			}

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 {
				break
			}

			// Log new information
			voters = append(voters, m.PubKeyEd)
			hashStr := hex.EncodeToString(m.BlockHash)
			counts[hashStr] += votes

			// If a block exceeds the vote threshold, we will end the loop.
			if counts[hashStr] > ctx.VoteLimit {
				timer.Stop()
				ctx.BlockHash = m.BlockHash
				return nil
			}
		}
	}
}

func processMsgReduction(ctx *Context, msg *payload.MsgReduction) (uint64, error) {
	// Verify message
	if !verifySignaturesReduction(ctx, msg) {
		return 0, nil
	}

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		// Either an old message or a malformed message
		return 0, nil
	}

	// Verify weight
	// TODO: implement

	// Make sure their score is valid, and calculate their amount of votes.
	votes, err := verifySortition(ctx, msg.Score, msg.PubKeyBLS, role, msg.Stake)
	if err != nil {
		return 0, err
	}

	if votes == 0 {
		return 0, nil
	}

	return votes, nil
}

func verifySignaturesReduction(ctx *Context, msg *payload.MsgReduction) bool {
	// Construct message
	var edMsg []byte
	edMsg = append(edMsg, msg.Score...)
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
