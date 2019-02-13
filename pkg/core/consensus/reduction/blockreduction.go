package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Block is the main function that runs during block reduction phase.
func Block(ctx *user.Context) error {
	// Set up a fallback value
	fallback := make([]byte, 32)

	// If we made it here without getting a block, set blockhash to fallback value.
	if ctx.BlockHash == nil {
		ctx.BlockHash = fallback
	}

	// Clear our votes out, so that we get a fresh set for this phase.
	ctx.BlockVotes = make([]*consensusmsg.Vote, 0)

	// Vote on passed block
	if err := blockVote(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countBlockVotes(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If BlockHash is nil, no clear winner was found within the time limit.
	// We will vote on the fallback value.
	if ctx.BlockHash == nil {
		ctx.BlockHash = fallback
	}

	// Vote on passed block
	if err := blockVote(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countBlockVotes(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If BlockHash is nil, no clear winner was found within the time limit.
	// So we will exit and restart the consensus.
	if ctx.BlockHash == nil {
		return nil
	}

	// If BlockHash is fallback, the committee has agreed to exit and restart
	// consensus.
	if bytes.Equal(ctx.BlockHash, fallback) {
		ctx.BlockHash = nil
		return nil
	}

	// If we did get a result, send block set agreement message
	if err := agreement.SendBlock(ctx); err != nil {
		return err
	}

	// Copy BlockVotes into SigSetVotes, in case we terminate a bit later
	// and BlockVotes will have been emptied.
	ctx.SigSetVotes = nil
	ctx.SigSetVotes = append(ctx.SigSetVotes, ctx.BlockVotes...)

	return nil
}

func blockVote(ctx *user.Context) error {
	// Set committee first
	size := user.CommitteeSize
	if len(ctx.Committee) < int(user.CommitteeSize) {
		size = uint8(len(ctx.Committee))
	}

	currentCommittee, err := sortition.CreateCommittee(ctx.Round, ctx.W, ctx.Step, size,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		return err
	}

	ctx.CurrentCommittee = currentCommittee

	// If we are not in the committee, then don't vote
	if votes := sortition.Verify(ctx.CurrentCommittee, []byte(*ctx.Keys.EdPubKey)); votes == 0 {
		return nil
	}

	// Sign block hash with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.BlockHash)
	if err != nil {
		return err
	}

	// Create reduction payload to gossip
	pl, err := consensusmsg.NewBlockReduction(ctx.BlockHash, sigBLS, ctx.Keys.BLSPubKey.Marshal())
	if err != nil {
		return err
	}

	// Sign the payload
	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		return err
	}

	// Create message
	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash, ctx.Step, sigEd,
		[]byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	// Gossip message
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	ctx.BlockReductionChan <- msg
	return nil
}

func countBlockVotes(ctx *user.Context) error {
	// Set vote limit
	voteLimit := uint8(len(ctx.CurrentCommittee))

	// Keep a counter of how many votes have been cast for a specific block
	counts := make(map[string]uint8)

	// Keep track of all nodes who have voted
	voters := make(map[string]bool)

	// Start the timer
	timer := time.NewTimer(user.StepTime * (time.Duration(ctx.Multiplier)))

	for {
		select {
		case <-timer.C:
			ctx.BlockHash = nil
			return nil
		case m := <-ctx.BlockReductionChan:
			pl := m.Payload.(*consensusmsg.BlockReduction)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's vote is already recorded
			if voters[pkEd] {
				break
			}

			// Get amount of votes
			votes := sortition.Verify(ctx.CurrentCommittee, m.PubKey)

			// Log new information
			voters[pkEd] = true
			hashStr := hex.EncodeToString(pl.BlockHash)
			counts[hashStr] += votes
			blockVote, err2 := consensusmsg.NewVote(pl.BlockHash, pl.PubKeyBLS, pl.SigBLS, ctx.Step)
			if err2 != nil {
				return err2
			}

			for i := uint8(0); i < votes; i++ {
				ctx.BlockVotes = append(ctx.BlockVotes, blockVote)
			}

			// If a block doesnt exceed the vote threshold, we keep going.
			if counts[hashStr] < voteLimit {
				break
			}

			timer.Stop()
			ctx.BlockHash = pl.BlockHash

			// We will also cut all the votes that did not vote for the winning block.
			for i, vote := range ctx.BlockVotes {
				if !bytes.Equal(vote.Hash, ctx.BlockHash) {
					ctx.BlockVotes = append(ctx.BlockVotes[:i], ctx.BlockVotes[i+1:]...)
				}
			}

			return nil
		}
	}
}
