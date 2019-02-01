package core

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Loop function for consensus. All functions are started as goroutines
// to be able to receive messages while the provisioner function is running.
func (b *Blockchain) segregatedByzantizeAgreement() {
	for {
		select {
		case <-b.roundChan:
			if b.generator {
				go b.blockGenerator()
			}

			if b.provisioner {
				go b.provision()
			}
		case m := <-b.consensusChan:
			go b.process(m)
		}
	}
}

// Block generator function
func (b *Blockchain) blockGenerator() {
	// First we reset our context values
	b.ctx.Reset()

	// Block generators do only one thing - generate blocks
	if err := consensus.GenerateBlock(b.ctx); err != nil {
		// Log
		b.generator = false
		return
	}
}

// Provisioner function
func (b *Blockchain) provision() {
	// First we reset our context values
	b.ctx.Reset()

	// Then, set the vote limit
	// Votes are currently quite similar to stake size (often a little bit less due
	// to the pseudo-random parts of sortition) so we will count with that for now
	var allVotes uint64
	for _, vote := range b.ctx.NodeWeights {
		allVotes += vote
	}

	// Should come out to around 75% of voting power
	b.ctx.VoteLimit = uint64(float64(allVotes) * 0.75)

	// Set up a channel that we can get agreement results from
	c := make(chan bool, 1)

	// Fire off the parallel block agreement phase
	go consensus.BlockAgreement(b.ctx, c)
out:
	for b.ctx.Step < consensus.MaxSteps {
		select {
		case v := <-c:
			// If it was false, something went wrong and we should quit
			if !v {
				// Log
				b.StopProvisioning()
				return
			}

			// If not, we proceed to the next phase
			break out
		default:
			// If this is the first step, or if we returned without a decisive vote,
			// collect blocks
			if b.ctx.BlockHash == nil {
				if err := consensus.BlockCollection(b.ctx); err != nil {
					// Log
					b.StopProvisioning()
					return
				}
			}

			// Vote on received block. The context object should hold a winning
			// block hash after this function returns.
			if err := consensus.BlockReduction(b.ctx); err != nil {
				// Log
				b.StopProvisioning()
				return
			}
		}
	}

	// Reset step counter
	b.ctx.Step = 1

	// Fire off parallel set agreement phase
	go consensus.SignatureSetAgreement(b.ctx, c)
	for b.ctx.Step < consensus.MaxSteps {
		select {
		case v := <-c:
			// If it was false, something went wrong and we should quit
			if !v {
				// Log
				b.StopProvisioning()
				return
			}

			// If not, we successfully terminate the consensus
			// Propagate the decided block if we have it
			for _, block := range b.ctx.CandidateBlocks {
				if bytes.Equal(block.Header.Hash, b.ctx.BlockHash) {
					// send block
				}
			}

			return
		default:
			// If this is the first step, or if we returned without a decisive vote,
			// collect signature sets
			if b.ctx.SigSetHash == nil {
				if err := consensus.SignatureSetGeneration(b.ctx); err != nil {
					// Log
					b.StopProvisioning()
					return
				}
			}

			// Vote on received signature set
			if err := consensus.SignatureSetReduction(b.ctx); err != nil {
				// Log
				b.StopProvisioning()
				return
			}
		}
	}
}

// Processor function for all incoming consensus messages.
// This function is implemented here to allow access to block verification
// during candidate collection, and to allow parallel processing of consensus
// messages, by splitting them according to their type and passing them
// to their respective channels.
func (b *Blockchain) process(m *payload.MsgConsensus) {
	if b.provisioner || b.generator {
		switch m.Payload.Type() {
		case consensusmsg.CandidateScoreID:
			b.ctx.CandidateScoreChan <- m
		case consensusmsg.CandidateID:
			// Verify the block first
			pl := m.Payload.(*consensusmsg.Candidate)
			if err := b.VerifyBlock(pl.Block); err != nil {
				// Log
				break
			}

			b.ctx.CandidateChan <- m
		case consensusmsg.ReductionID:
			b.ctx.ReductionChan <- m
		case consensusmsg.SetAgreementID:
			b.ctx.SetAgreementChan <- m
		case consensusmsg.SigSetCandidateID:
			b.ctx.SigSetCandidateChan <- m
		case consensusmsg.SigSetVoteID:
			b.ctx.SigSetVoteChan <- m
		}
	}
}

// StartProvisioning will set the node to provisioner status,
// and will start participating in block reduction and binary agreement
// phases of the protocol.
func (b *Blockchain) StartProvisioning() error {
	if b.provisioner {
		return errors.New("already provisioning")
	}

	keys, err := consensus.NewRandKeys()
	if err != nil {
		return err
	}

	ctx, err := consensus.NewContext(b.tau, b.bidWeight, b.totalStakeWeight, b.round,
		b.currSeed, b.net, keys)
	if err != nil {
		return err
	}

	b.ctx = ctx
	if err := b.SetupProvisioners(); err != nil {
		return err
	}

	b.provisioner = true
	return nil
}

// StopProvisioning will stop the provisioning process
func (b *Blockchain) StopProvisioning() {
	b.provisioner = false
	b.ctx.Clear()
	b.provisioners = nil
}
