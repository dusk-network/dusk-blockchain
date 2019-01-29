package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	errNoBlockchainDb    = errors.New("Blockchain database is not available")
	errInitialisedCheck  = errors.New("Failed to check if blockchain db is already initialised")
	errBlockValidation   = errors.New("Block failed sanity check")
	errBlockVerification = errors.New("Block failed to be consistent with the current blockchain")
)

const maxLockTime = math.MaxUint16

// Blockchain defines a processor for blocks and transactions which
// makes sure that any passed data is in line with the current
// consensus, and maintains a memory pool of all known unconfirmed
// transactions. Properly verified transactions and blocks will be
// added to the memory pool and the database respectively.
type Blockchain struct {
	// Basic fields
	memPool *MemPool
	net     protocol.Magic
	height  uint64
	db      *database.BlockchainDB

	// Consensus related
	currSeed      []byte                     // Seed of the current round of consensus
	round         uint64                     // Current round (block height + 1)
	tau           uint64                     // Current generator threshold
	lastHeader    *block.Header              // Last validated block on the chain
	roundChan     chan int                   // Channel used to signify start of a new round
	ctx           *consensus.Context         // Consensus context object
	consensusChan chan *payload.MsgConsensus // Channel for consensus messages

	// Block generator related fields
	generator bool
	bidWeight uint64

	// Provisioner related fields
	provisioner      bool
	stakeWeight      uint64 // The amount of DUSK staked by the node
	totalStakeWeight uint64 // The total amount of DUSK staked
	provisioners     Provisioners
}

// NewBlockchain returns a new Blockchain instance with an initialized mempool.
// This Blockchain instance should then be ready to process incoming transactions and blocks.
func NewBlockchain(net protocol.Magic) (*Blockchain, error) {
	db := database.GetInstance()

	marker := []byte("HasBeenInitialisedAlready")
	init, err := db.Has(marker)
	if err != nil {
		return nil, errInitialisedCheck
	}

	if !init {
		// This is a new db, so initialise it
		log.WithField("prefix", "blockchain").Info("New Blockchain database initialisation")
		db.Put(marker, []byte{})

		// Add Genesis block (No transactions in Genesis block)
		if net == protocol.DevNet {
			genesisBlock, err := hex.DecodeString(GenesisBlock)
			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}

			r := bytes.NewReader(genesisBlock)
			b := block.NewBlock()
			if err := b.Decode(r); err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}

			err = db.WriteHeaders([]*block.Header{b.Header})
			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header")
				db.Delete(marker)
				return nil, err
			}
		}

		if net == protocol.TestNet {
			fmt.Println("TODO: Setup the genesisBlock for TestNet")
			return nil, nil
		}

		if net == protocol.MainNet {
			fmt.Println("TODO: Setup the genesisBlock for MainNet")
			return nil, nil
		}
	}

	chain := &Blockchain{}

	// Set up mempool and populate struct fields
	chain.memPool = &MemPool{}
	chain.memPool.Init()
	chain.db = db
	chain.net = net

	// Consensus set-up
	chain.consensusChan = make(chan *payload.MsgConsensus, 500)
	chain.roundChan = make(chan int, 1)
	chain.lastHeader, err = chain.GetLatestHeader()
	if err != nil {
		return nil, err
	}

	chain.height = chain.lastHeader.Height
	chain.round = chain.height + 1
	chain.currSeed = chain.lastHeader.Seed

	// Start consensus loop
	go chain.segregatedByzantizeAgreement()
	return chain, nil
}

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

	// Set up a channel that we can get agreement results from
	c := make(chan bool, 1)

	// Collect a block, and start the loop
	if err := consensus.BlockCollection(b.ctx); err != nil {
		// Log
		b.StopProvisioning()
		return
	}

	for b.ctx.Step < consensus.MaxSteps {
		// Fire off the parallel block agreement phase
		go consensus.BlockAgreement(b.ctx, c)
	out:
		for {
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
				// Vote on received block. The context object should hold a winning
				// block hash after this function returns.
				if err := consensus.BlockReduction(b.ctx); err != nil {
					// Log
					b.StopProvisioning()
					return
				}
			}
		}

		for b.ctx.Step < consensus.MaxSteps {
			// Fire off parallel set agreement phase
			go consensus.SignatureSetAgreement(b.ctx, c)
			for {
				select {
				case v := <-c:
					// If it was false, something went wrong and we should quit
					if !v {
						// Log
						b.StopProvisioning()
						return
					}

					// If not, we successfully terminate the consensus
					// send block
					return
				default:
					// Generate our signature set candidate and collect the best one
					if err := consensus.SignatureSetGeneration(b.ctx); err != nil {
						// Log
						b.StopProvisioning()
						return
					}

					// Vote on received signature set.
					if err := consensus.SignatureSetReduction(b.ctx); err != nil {
						// Log
						b.StopProvisioning()
						return
					}
				}
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

// GetLatestHeader gives the latest block header
func (b *Blockchain) GetLatestHeader() (*block.Header, error) {
	db := database.GetInstance()

	return db.GetLatestHeader()
}

// GetHeaders gives block headers from the database, starting and
// stopping at the provided locators.
func (b *Blockchain) GetHeaders(start []byte, stop []byte) ([]*block.Header, error) {
	db := database.GetInstance()

	return db.ReadHeaders(start, stop)
}

// GetBlock will return the block from the received hash
func (b *Blockchain) GetBlock(hash []byte) (*block.Block, error) {
	bd := database.GetInstance()

	return bd.GetBlock(hash)
}

// AddHeaders will add block headers to the chain.
func (b *Blockchain) AddHeaders(msg *payload.MsgHeaders) error {
	db := database.GetInstance()
	if err := db.WriteHeaders(msg.Headers); err != nil {
		return err
	}
	return nil
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

	ctx, err := consensus.NewContext(b.tau, b.totalStakeWeight, b.round, b.currSeed, b.net, keys)
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
