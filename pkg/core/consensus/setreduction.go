package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// SignatureSetReduction is the signature set reduction phase of the consensus.
func SignatureSetReduction(ctx *Context, c chan *payload.MsgSignatureSet,
	v chan *payload.MsgSigSetVote) error {
	for ctx.step = 1; ctx.step < maxSteps; ctx.step++ {
		// Generate our own signature set and propagate
		if err := signatureSetGeneration(ctx); err != nil {
			return err
		}

		// Collect signature set with highest score
		if err := signatureSetCollection(ctx, c); err != nil {
			return err
		}

		// Vote on collected signature set
		if err := committeeVoteSigSet(ctx); err != nil {
			return err
		}

		// Receive all other votes
		if err := countVotesSigSet(ctx, v); err != nil {
			return err
		}

		// If we timed out, go back to the beginning of the loop
		if ctx.SignatureSet == nil {
			continue
		}

		ctx.step++
		if err := committeeVoteSigSet(ctx); err != nil {
			return err
		}

		if err := countVotesSigSet(ctx, v); err != nil {
			return err
		}

		// If we got a result, terminate
		if ctx.SignatureSet != nil {
			return nil
		}
	}

	return nil
}

func signatureSetCollection(ctx *Context, c chan *payload.MsgSignatureSet) error {
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	highest := ctx.weight
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			return nil
		case m := <-c:
			// Check if this node's signature set is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKey) {
					break out
				}
			}

			// Verify the message
			if !verifyMsgSignatureSet(ctx, m) {
				break out
			}

			// If the stake is higher than our current one, replace
			if m.Stake > highest {
				highest = m.Stake
				ctx.SignatureSet = m.SignatureSet
			}
		}
	}
}

func signatureSetGeneration(ctx *Context) error {
	sigs, _ := crypto.RandEntropy(200) // Placeholder
	ctx.SignatureSet = sigs

	// Construct message
	var edMsg []byte
	binary.LittleEndian.PutUint64(edMsg, ctx.weight)
	binary.LittleEndian.PutUint64(edMsg, ctx.Round)
	edMsg = append(edMsg, ctx.LastHeader.Hash...)
	edMsg = append(edMsg, ctx.BlockHash...)
	edMsg = append(edMsg, sigs...)

	// Sign with ed25519
	edSig := ctx.EDSign(ctx.Keys.EdSecretKey, edMsg)

	// Create message to gossip
	msg, err := payload.NewMsgSignatureSet(ctx.weight, ctx.Round, ctx.LastHeader.Hash, ctx.BlockHash,
		ctx.SignatureSet, edSig, []byte(*ctx.Keys.EdPubKey))
	if err != nil {
		return err
	}

	// Gossip msg
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	return nil
}

func verifyMsgSignatureSet(ctx *Context, msg *payload.MsgSignatureSet) bool {
	if msg.Round != ctx.Round {
		return false
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return false
	}

	// Verify weight
	// TODO: implement

	// Construct msg to verify
	var edMsg []byte
	binary.LittleEndian.PutUint64(edMsg, msg.Stake)
	binary.LittleEndian.PutUint64(edMsg, msg.Round)
	edMsg = append(edMsg, msg.PrevBlockHash...)
	edMsg = append(edMsg, msg.WinningBlockHash...)
	edMsg = append(edMsg, msg.SignatureSet...)

	// Verify signature
	return ctx.EDVerify(msg.PubKey, edMsg, msg.Signature)
}

func committeeVoteSigSet(ctx *Context) error {
	// Sign signature set with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.SignatureSet)
	if err != nil {
		return err
	}

	// Construct message
	var edMsg []byte
	binary.LittleEndian.PutUint64(edMsg, ctx.weight)
	binary.LittleEndian.PutUint64(edMsg, ctx.Round)
	edMsg = append(edMsg, byte(ctx.step))
	edMsg = append(edMsg, ctx.LastHeader.Hash...)
	edMsg = append(edMsg, ctx.SignatureSet...)
	edMsg = append(edMsg, sigBLS...)

	// Sign with ed25519
	sigEd := ctx.EDSign(ctx.Keys.EdSecretKey, edMsg)

	// Create signature set vote message to gossip
	blsPubBytes, err := ctx.Keys.BLSPubKey.MarshalBinary()
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgSigSetVote(ctx.weight, ctx.Round, ctx.step, ctx.LastHeader.Hash,
		ctx.SignatureSet, sigBLS, sigEd, blsPubBytes, []byte(*ctx.Keys.EdSecretKey))
	if err != nil {
		return err
	}

	// Gossip message
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	return nil
}

func countVotesSigSet(ctx *Context, v chan *payload.MsgSigSetVote) error {
	counts := make(map[string]uint64)
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	counts[hex.EncodeToString(ctx.SignatureSet)] += ctx.weight
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			ctx.SignatureSet = nil
			return nil
		case m := <-v:
			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKeyEd) {
					break out
				}
			}

			// Verify the message
			if !verifyMsgSigSetVote(ctx, m) {
				break out
			}

			voters = append(voters, m.PubKeyEd)
			setStr := hex.EncodeToString(m.SignatureSet)
			counts[setStr] += m.Stake

			// If a set exceeds vote threshold, we will end the loop.
			if counts[setStr] > ctx.VoteLimit {
				timer.Stop()
				ctx.SignatureSet = m.SignatureSet
				return nil
			}
		}
	}
}

func verifyMsgSigSetVote(ctx *Context, msg *payload.MsgSigSetVote) bool {
	// Synchrony checks
	if msg.Round != ctx.Round {
		return false
	}

	if msg.Step != ctx.step {
		return false
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return false
	}

	// Verify weight
	// TODO: implement

	// Verify signatures

	// Check BLS
	if err := ctx.BLSVerify(msg.PubKeyBLS, msg.SignatureSet, msg.SigBLS); err != nil {
		return false
	}

	// Check ed25519
	// Construct message
	var edMsg []byte
	binary.LittleEndian.PutUint64(edMsg, msg.Stake)
	binary.LittleEndian.PutUint64(edMsg, msg.Round)
	edMsg = append(edMsg, byte(msg.Step))
	edMsg = append(edMsg, msg.PrevBlockHash...)
	edMsg = append(edMsg, msg.SignatureSet...)
	edMsg = append(edMsg, msg.SigBLS...)

	// Verify ed25519
	return ctx.EDVerify(msg.PubKeyEd, edMsg, msg.SigEd)
}
