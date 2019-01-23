package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// BinaryAgreement is the main function that runs during the binary agreement
// phase of the consensus.
func BinaryAgreement(ctx *Context) error {
	// Prepare empty block
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		return err
	}

	for ctx.step = 1; ctx.step < maxSteps; ctx.step++ {
		// Save our currently kept block hash
		var startHash []byte
		startHash = append(startHash, ctx.BlockHash...)

		var msgs []*payload.MsgConsensus
		var err error
		if _, err := committeeVoteBinary(ctx); err != nil {
			return err
		}

		_, err = countVotesBinary(ctx)
		if err != nil {
			return err
		}

		// Coin-flipped-to-0 step
		if ctx.BlockHash == nil {
			ctx.BlockHash = startHash
		}

		if ctx.Empty {
			ctx.BlockHash = emptyBlock.Header.Hash
		} else if ctx.step == 1 {
			ctx.step = maxSteps
			// ctx.RaiseVoteLimit()
			if _, err := committeeVoteBinary(ctx); err != nil {
				return err
			}

			return nil
		}

		ctx.step++
		if _, err := committeeVoteBinary(ctx); err != nil {
			return err
		}

		_, err = countVotesBinary(ctx)

		// Coin-flipped-to-1 step
		if ctx.BlockHash == nil {
			ctx.BlockHash = emptyBlock.Header.Hash
			ctx.Empty = true
		}

		if ctx.Empty {
			return nil
		}

		ctx.step++
		vote, err := committeeVoteBinary(ctx)
		if err != nil {
			return err
		}

		msgs, err = countVotesBinary(ctx)
		msgs = append(msgs, vote)

		// CommonCoin step
		if ctx.BlockHash == nil {
			result, err := commonCoin(ctx, msgs)
			if err != nil {
				return err
			}

			if result == 0 {
				ctx.BlockHash = startHash
				continue
			}

			ctx.BlockHash = emptyBlock.Header.Hash
		}
	}

	return nil
}

func commonCoin(ctx *Context, allVotes []*payload.MsgConsensus) (uint64, error) {
	var lenHash, _ = new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
	for i, vote := range allVotes {
		// Don't check for validity, as it has already been checked during vote count
		_, votes, err := processMsg(ctx, vote)
		if err != nil {
			return 0, err
		}

		pl := vote.Payload.(*consensusmsg.Agreement)
		for j := uint64(1); j < votes; j++ {
			binary.LittleEndian.PutUint32(pl.BlockHash, uint32(i))
			result, err := hash.Sha3256(pl.BlockHash)
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

func committeeVoteBinary(ctx *Context) (*payload.MsgConsensus, error) {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	if err := sortition(ctx, role); err != nil {
		return nil, err
	}

	if ctx.votes > 0 {
		// Sign block hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return nil, err
		}

		// Create agreement payload to gossip
		blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
		pl, err := consensusmsg.NewAgreement(ctx.Score, ctx.Empty, ctx.step, ctx.BlockHash,
			sigBLS, blsPubBytes)
		if err != nil {
			return nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		if err != nil {
			return nil, err
		}

		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			sigEd, []byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return nil, err
		}

		if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
			return nil, err
		}

		// Return the message for inclusion in the common coin procedure
		return msg, nil
	}

	return nil, nil
}

func countVotesBinary(ctx *Context) ([]*payload.MsgConsensus, error) {
	counts := make(map[string]uint64)
	var voters [][]byte
	var allMsgs []*payload.MsgConsensus
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	counts[hex.EncodeToString(ctx.BlockHash)] += ctx.votes
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			ctx.BlockHash = nil
			return allMsgs, nil
		case m := <-ctx.msgs:
			// Check first off if this message is the right one, if not
			// we discard it.
			if m.ID != consensusmsg.AgreementID {
				break
			}

			pl := m.Payload.(*consensusmsg.Agreement)

			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKey) {
					break out
				}
			}

			// Verify the message score and get back it's contents
			valid, votes, err := processMsg(ctx, m)
			if err != nil {
				return nil, err
			}

			if !valid {
				break
			}

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 {
				break
			}

			// Log new information
			voters = append(voters, m.PubKey)
			hashStr := hex.EncodeToString(pl.BlockHash)
			counts[hashStr] += votes

			// Save vote for common coin
			allMsgs = append(allMsgs, m)

			// If a block exceeds the vote threshold, we will end the loop.
			if counts[hashStr] > ctx.VoteLimit {
				timer.Stop()
				ctx.Empty = pl.Empty
				ctx.BlockHash = pl.BlockHash
				return allMsgs, nil
			}
		}
	}
}
