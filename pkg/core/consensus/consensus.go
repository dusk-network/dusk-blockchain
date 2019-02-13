package consensus

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/collection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

type Config struct {
	net protocol.Magic
	// Consensus related
	currSeed      []byte                     // Seed of the current round of consensus
	round         uint64                     // Current round (block height + 1)
	tau           uint64                     // Current generator threshold
	lastHeader    *block.Header              // Last validated block on the chain
	RoundChan     chan int                   // Channel used to signify start of a new round
	ctx           *user.Context              // Consensus context object
	ConsensusChan chan *payload.MsgConsensus // Channel for consensus messages

	// Block generator related fields
	generator bool
	bidWeight uint64

	// Provisioner related fields
	provisioner      bool
	stakeWeight      uint64 // The amount of DUSK staked by the node
	totalStakeWeight uint64 // The total amount of DUSK staked
	provisioners     Provisioners

	GetLatestHeight        func() uint64                      // Gets the latest height
	GetBlockHeaderByHeight func(uint64) (block.Header, error) // Gets a block Header using it's height
	GetBlock               func([]byte) (block.Block, error)  // Gets a block  using the hash of the header
	VerifyBlock            func(*block.Block) error           // Checks whether a block is valid or not

}

type Consensus struct {
	*Config
}

func New(cfg *Config) (*Consensus, error) {
	if cfg.GetLatestHeight == nil {
		return nil, errors.New("pointer to getLatestHeight function is nil ")
	}
	if cfg.GetBlockHeaderByHeight == nil {
		return nil, errors.New("pointer to getBlockHeaderByHeight function is nil ")
	}
	if cfg.GetBlock == nil {
		return nil, errors.New("pointer to GetBlock function is nil ")
	}
	if cfg.VerifyBlock == nil {
		return nil, errors.New("pointer to VerifyBlock function is nil ")
	}

	cfg.ConsensusChan = make(chan *payload.MsgConsensus, 500)
	cfg.RoundChan = make(chan int, 1)

	latestHeight := cfg.GetLatestHeight()
	latestHeader, err := cfg.GetBlockHeaderByHeight(latestHeight)
	if err != nil {
		return nil, err
	}

	c := &Consensus{cfg}

	c.round = latestHeight + 1
	c.currSeed = latestHeader.Seed

	// Start consensus loop
	go c.segregatedByzantizeAgreement()

	return c, nil
}

// Loop function for consensus.
func (c *Consensus) segregatedByzantizeAgreement() {
	for {
		select {
		case <-c.RoundChan:
			go c.consensus()
		case m := <-c.ConsensusChan:
			go c.process(m)
		}
	}
}

func (c *Consensus) consensus() {
	// First we reset our context values
	c.ctx.Reset()

	if c.generator {
		// Generate block and score, and propagate
		if err := generation.Block(c.ctx); err != nil {
			// Log
			c.generator = false
			return
		}
	}

	if err := collection.Block(c.ctx); err != nil {
		// Log
		c.StopProvisioning()
		return
	}

	// Set up a channel that we can get agreement results from
	ch := make(chan bool, 1)

	// Fire off the parallel block agreement phase if we're provisioning
	if c.provisioner {
		go agreement.Block(c.ctx, ch)
	}

	// Block inner loop
	for c.ctx.Step < user.MaxSteps {
		select {
		case v := <-ch:
			// If it was false, something went wrong and we should quit
			if !v {
				// Log
				c.StopProvisioning()
				return
			}

			// If not, we proceed to the next phase by maxing out the
			// step counter.
			c.ctx.Step = user.MaxSteps
		default:
			// Vote on received block. The context object should hold a winning
			// block hash after this function returns.
			if err := reduction.Block(c.ctx); err != nil {
				// Log
				c.StopProvisioning()
				return
			}

			if c.ctx.BlockHash != nil {
				continue
			}

			// If we did not get a result, increase the multiplier and
			// exit the loop.
			c.ctx.Step = user.MaxSteps
			if c.ctx.Multiplier < 10 {
				c.ctx.Multiplier = c.ctx.Multiplier * 2
			}
		}
	}

	// If we did not get a result, restart the consensus from block generation.
	if c.ctx.BlockHash == nil {
		c.RoundChan <- 1
		return
	}

	// Block generators don't need to keep up after this point
	if !c.provisioner {
		return
	}

	// Reset step counter
	c.ctx.Step = 1

	// Fire off parallel set agreement phase
	go agreement.SignatureSet(c.ctx, ch)

	// Signature set inner loop
	for c.ctx.Step < user.MaxSteps {
		select {
		case v := <-ch:
			// If it was false, something went wrong and we should quit
			if !v {
				// Log
				c.StopProvisioning()
				return
			}

			// If not, we successfully terminate the consensus
			// Propagate the decided block if we have it
			for _, block := range c.ctx.CandidateBlocks {
				if bytes.Equal(block.Header.Hash, c.ctx.BlockHash) {
					// send block
				}
			}

			return
		default:
			// If this is the first step, or if we returned without a decisive vote,
			// collect signature sets
			if c.ctx.SigSetHash == nil {
				if err := generation.SignatureSet(c.ctx); err != nil {
					// Log
					c.StopProvisioning()
					return
				}

				if err := collection.SignatureSet(c.ctx); err != nil {
					// Log
					c.StopProvisioning()
					return
				}
			}

			// Vote on received signature set
			if err := reduction.SignatureSet(c.ctx); err != nil {
				// Log
				c.StopProvisioning()
				return
			}

			// Increase multiplier
			if c.ctx.Multiplier < 10 {
				c.ctx.Multiplier = c.ctx.Multiplier * 2
			}
		}
	}

	// Reset multiplier
	c.ctx.Multiplier = 1
}

func (c *Consensus) StartGenerating() error {
	if c.generator {
		return errors.New("already generating")
	}

	keys, err := user.NewRandKeys()
	if err != nil {
		return err
	}

	height := c.GetLatestHeight()

	ctx, err := user.NewContext(c.tau, c.bidWeight, c.totalStakeWeight, height+1,
		c.lastHeader.Seed, c.net, keys)
	if err != nil {
		return err
	}

	c.ctx = ctx
	if err := c.SetupProvisioners(); err != nil {
		return err
	}

	c.generator = true
	return nil
}

func (c *Consensus) StopGenerating() {
	c.generator = false
	c.ctx.Clear()
	c.provisioners = nil
}

// StartProvisioning will set the node to provisioner status,
// and will start participating in block reduction and binary agreement
// phases of the protocol.
func (c *Consensus) StartProvisioning() error {
	if c.provisioner {
		return errors.New("already provisioning")
	}

	keys, err := user.NewRandKeys()
	if err != nil {
		return err
	}

	height := c.GetLatestHeight()

	ctx, err := user.NewContext(c.tau, c.bidWeight, c.totalStakeWeight, height+1,
		c.lastHeader.Seed, c.net, keys)
	if err != nil {
		return err
	}

	c.ctx = ctx
	if err := c.SetupProvisioners(); err != nil {
		return err
	}

	c.provisioner = true
	return nil
}

// StopProvisioning will stop the provisioning process
func (c *Consensus) StopProvisioning() {
	c.provisioner = false
	c.ctx.Clear()
	c.provisioners = nil
}

func (c *Consensus) process(m *payload.MsgConsensus) {
	if c.provisioner || c.generator {
		switch m.Payload.Type() {
		case consensusmsg.CandidateID:
			// Verify the block first
			pl := m.Payload.(*consensusmsg.Candidate)
			if err := c.VerifyBlock(pl.Block); err != nil {
				// Log
				break
			}

			err := msg.Process(c.ctx, m)
			if err != nil && err.Priority == prerror.High {
				// Send something to error channel
				// If error is low priority it's simply an invalid message,
				// and we don't need to handle it.
			}
		default:
			err := msg.Process(c.ctx, m)
			if err != nil && err.Priority == prerror.High {
				// Send something to error channel
			}
		}
	}
}

func (c *Consensus) UpdateProvisioners(blk *block.Block) error {
	return nil
}
