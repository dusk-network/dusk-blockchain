package core

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/collection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Loop function for consensus.
func (b *Blockchain) segregatedByzantizeAgreement() {
	for {
		select {
		case <-b.roundChan:
			go b.consensus()
		case m := <-b.consensusChan:
			go b.process(m)
		}
	}
}

func (b *Blockchain) consensus() {
	// First we reset our context values
	b.ctx.Reset()

	if b.generator {
		// Generate block and score, and propagate
		if err := generation.Block(b.ctx); err != nil {
			// Log
			b.generator = false
			return
		}
	}

	if err := collection.Block(b.ctx); err != nil {
		// Log
		b.StopProvisioning()
		return
	}

	// Set up a channel that we can get agreement results from
	c := make(chan bool, 1)

	// Fire off the parallel block agreement phase if we're provisioning
	if b.provisioner {
		go agreement.Block(b.ctx, c)
	}

	// Block inner loop
	for b.ctx.Step < user.MaxSteps {
		select {
		case v := <-c:
			// If it was false, something went wrong and we should quit
			if !v {
				// Log
				b.StopProvisioning()
				return
			}

			// If not, we proceed to the next phase by maxing out the
			// step counter.
			b.ctx.Step = user.MaxSteps
		default:
			// Vote on received block. The context object should hold a winning
			// block hash after this function returns.
			if err := reduction.Block(b.ctx); err != nil {
				// Log
				b.StopProvisioning()
				return
			}

			if b.ctx.BlockHash != nil {
				continue
			}

			// If we did not get a result, increase the multiplier and
			// exit the loop.
			b.ctx.Step = user.MaxSteps
			if b.ctx.Multiplier < 10 {
				b.ctx.Multiplier = b.ctx.Multiplier * 2
			}
		}
	}

	// If we did not get a result, restart the consensus from block generation.
	if b.ctx.BlockHash == nil {
		b.roundChan <- 1
		return
	}

	// Block generators don't need to keep up after this point
	if !b.provisioner {
		return
	}

	// Reset step counter
	b.ctx.Step = 1

	// Fire off parallel set agreement phase
	go agreement.SignatureSet(b.ctx, c)

	// Signature set inner loop
	for b.ctx.Step < user.MaxSteps {
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
				if err := generation.SignatureSet(b.ctx); err != nil {
					// Log
					b.StopProvisioning()
					return
				}

				if err := collection.SignatureSet(b.ctx); err != nil {
					// Log
					b.StopProvisioning()
					return
				}
			}

			// Vote on received signature set
			if err := reduction.SignatureSet(b.ctx); err != nil {
				// Log
				b.StopProvisioning()
				return
			}

			// Increase multiplier
			if b.ctx.Multiplier < 10 {
				b.ctx.Multiplier = b.ctx.Multiplier * 2
			}
		}
	}

	// Reset multiplier
	b.ctx.Multiplier = 1
}

// Processor function for all incoming consensus messages.
// This function is implemented here to allow access to block verification
// during candidate collection, and to allow parallel processing of consensus
// messages, by splitting them according to their type and passing them
// to their respective channels.
func (b *Blockchain) process(m *payload.MsgConsensus) {
	if b.provisioner || b.generator {
		switch m.Payload.Type() {
		case consensusmsg.CandidateID:
			// Verify the block first
			pl := m.Payload.(*consensusmsg.Candidate)
			if err := b.VerifyBlock(pl.Block); err != nil {
				// Log
				break
			}

			err := msg.Process(b.ctx, m)
			if err != nil && err.Priority == prerror.High {
				// Send something to error channel
				// If error is low priority it's simply an invalid message,
				// and we don't need to handle it.
			}
		default:
			err := msg.Process(b.ctx, m)
			if err != nil && err.Priority == prerror.High {
				// Send something to error channel
			}
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

	keys, err := user.NewRandKeys()
	if err != nil {
		return err
	}

	ctx, err := user.NewContext(0, b.bidWeight, b.totalStakeWeight, b.height+1,
		b.lastHeader.Seed, b.net, keys)
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
