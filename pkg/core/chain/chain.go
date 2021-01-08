// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logger.WithFields(logger.Fields{"process": "chain"})

// Verifier performs checks on the blockchain and potentially new incoming block
type Verifier interface {
	// PerformSanityCheck on first N blocks and M last blocks
	PerformSanityCheck(startAt uint64, firstBlocksAmount uint64, lastBlockAmount uint64) error
	// SanityCheckBlock will verify whether a block is valid according to the rules of the consensus
	SanityCheckBlock(prevBlock block.Block, blk block.Block) error
}

// Loader is an interface which abstracts away the storage used by the Chain to
// store the blockchain
type Loader interface {
	// LoadTip of the chain
	LoadTip() (*block.Block, error)
	// Clear removes everything from the DB
	Clear() error
	// Close the Loader and finalizes any pending connection
	Close(driver string) error
	// Height returns the current height as stored in the loader
	Height() (uint64, error)
	// BlockAt returns the block at a given height
	BlockAt(uint64) (block.Block, error)
	// Append a block on the storage
	Append(*block.Block) error
}

// Ledger is the Chain interface used in tests
type Ledger interface {
	CurrentHeight() uint64
	ProcessSucceedingBlock(block.Block)
	ProcessSyncBlock(block.Block) error
	ProduceBlock() error
	StopBlockProduction()
}

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB

	// loader abstracts away the persistence aspect of Block operations
	loader Loader

	// verifier performs verifications on the block
	verifier Verifier

	// current blockchain tip of local state
	lock sync.RWMutex
	tip  *block.Block

	// Current set of provisioners
	p *user.Provisioners

	// Consensus loop
	loop           *loop.Consensus
	CatchBlockChan chan consensus.Results

	// rusk client
	proxy transactions.Proxy

	ctx context.Context
}

// New returns a new chain object. It accepts the EventBus (for messages coming
// from (remote) consensus components, the RPCBus for dispatching synchronous
// data related to Certificates, Blocks, Rounds and progress.
func New(ctx context.Context, db database.DB, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, loader Loader, verifier Verifier, srv *grpc.Server, proxy transactions.Proxy, loop *loop.Consensus) (*Chain, error) {
	chain := &Chain{
		eventBus:       eventBus,
		rpcBus:         rpcBus,
		db:             db,
		loader:         loader,
		verifier:       verifier,
		proxy:          proxy,
		ctx:            ctx,
		loop:           loop,
		CatchBlockChan: make(chan consensus.Results),
	}

	provisioners, err := proxy.Executor().GetProvisioners(ctx)
	if err != nil {
		log.WithError(err).Error("Error in getting provisioners")
		return nil, err
	}
	chain.p = &provisioners

	prevBlock, err := loader.LoadTip()
	if err != nil {
		return nil, err
	}
	chain.tip = prevBlock

	if prevBlock.Header.Height == 0 {
		// TODO: this is currently mocking bid values, and should be removed when
		// RUSK integration is finished, and testnet is ready to launch.
		if errV := setupBidValues(chain.db); errV != nil {
			return nil, errV
		}

		if config.Get().Genesis.Legacy {
			if errV := ReconstructCommittee(chain.p, prevBlock); errV != nil {
				return nil, errV
			}
		}
	}

	if srv != nil {
		node.RegisterChainServer(srv, chain)
	}

	return chain, nil
}

// CurrentHeight returns the height of the chain tip.
func (c *Chain) CurrentHeight() uint64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.tip.Header.Height
}

// GetRoundUpdate returns the current RoundUpdate
func (c *Chain) GetRoundUpdate() consensus.RoundUpdate {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.getRoundUpdate()
}

// StopBlockProduction notifies the loop that it must return immediately. It
// does that by pushing an error through the Block Result channel
func (c *Chain) StopBlockProduction() {
	// Kill the `ProduceBlock` goroutine.
	select {
	case c.CatchBlockChan <- consensus.Results{Blk: block.Block{}, Err: errors.New("syncing mode started")}:
	default:
	}
}

// ProduceBlock ...
func (c *Chain) ProduceBlock() error {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	for {
		candidate := c.produceBlock(ctx)
		block, err := candidate.Blk, candidate.Err
		if err != nil {
			return err
		}

		// Otherwise, accept the block directly.
		if !block.IsEmpty() {
			if err = c.AcceptSuccessiveBlock(block); err != nil {
				return err
			}
		}
	}
}

func (c *Chain) produceBlock(ctx context.Context) (winner consensus.Results) {
	ru := c.GetRoundUpdate()

	if c.loop != nil {
		scr, agr, err := c.loop.CreateStateMachine(c.db, config.ConsensusTimeOut, c.VerifyCandidateBlock, c.CatchBlockChan)
		if err != nil {
			// TODO: errors should be handled by the caller
			log.WithError(err).Error("could not create consensus state machine")
			winner.Err = err
			return winner
		}

		return c.loop.Spin(ctx, scr, agr, ru)
	}

	for {
		select {
		case r := <-c.CatchBlockChan:
			if r.Blk.Header != nil && r.Blk.Header.Height != ru.Round {
				continue
			}

			winner = r
			return
		case <-ctx.Done():
			return
		}
	}
}

// ProcessSucceedingBlock will handle blocks incoming from the network,
// which directly succeed the known chain tip.
func (c *Chain) ProcessSucceedingBlock(blk block.Block) {
	log.WithField("height", blk.Header.Height).Trace("received succeeding block")

	select {
	case c.CatchBlockChan <- consensus.Results{Blk: blk, Err: nil}:
	default:
	}
}

// ProcessSyncBlock will handle blocks which are received through a
// synchronization procedure.
func (c *Chain) ProcessSyncBlock(blk block.Block) error {
	log.WithField("height", blk.Header.Height).Trace("received sync block")
	return c.AcceptBlock(blk)
}

// AcceptSuccessiveBlock will accept a block which directly follows the chain
// tip, and advertises it to the node's peers.
func (c *Chain) AcceptSuccessiveBlock(blk block.Block) error {
	log.WithField("height", blk.Header.Height).Trace("accepting succeeding block")
	if err := c.AcceptBlock(blk); err != nil {
		return err
	}

	log.Trace("gossiping block")
	if err := c.advertiseBlock(blk); err != nil {
		log.WithError(err).Error("block advertising failed")
		return err
	}

	return nil
}

// AcceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and stateful checks are true
// Returns nil, if checks passed and block was successfully saved
func (c *Chain) AcceptBlock(blk block.Block) error {
	field := logger.Fields{"process": "accept block", "height": blk.Header.Height}
	l := log.WithFields(field)

	// Guard the c.tip field
	c.lock.Lock()
	defer c.lock.Unlock()

	l.Trace("verifying block")
	// 1. Check that stateless and stateful checks pass
	if err := c.verifier.SanityCheckBlock(*c.tip, blk); err != nil {
		l.WithError(err).Error("block verification failed")
		return err
	}

	// 2. Check the certificate
	// This check should avoid a possible race condition between accepting two blocks
	// at the same height, as the probability of the committee creating two valid certificates
	// for the same round is negligible.
	l.Trace("verifying block certificate")
	if err := verifiers.CheckBlockCertificate(*c.p, blk); err != nil {
		l.WithError(err).Error("certificate verification failed")
		return err
	}

	// 3. Call ExecuteStateTransitionFunction
	prov_num := c.p.Set.Len()
	l.WithField("provisioners", prov_num).Info("calling ExecuteStateTransitionFunction")

	// TODO: the context here should maybe used to set a timeout
	provisioners, err := c.proxy.Executor().ExecuteStateTransition(c.ctx, blk.Txs, blk.Header.Height)
	if err != nil {
		l.WithError(err).Error("Error in executing the state transition")
		return err
	}

	// Update the provisioners as blk.Txs may bring new provisioners to the current state
	c.p = &provisioners
	c.tip = &blk

	l.WithField("provisioners", c.p.Set.Len()).
		WithField("added", c.p.Set.Len()-prov_num).
		Info("after ExecuteStateTransitionFunction")

	if config.Get().API.Enabled {
		go c.storeStakesInStormDB(blk.Header.Height)
	}

	// 4. Store the approved block
	l.Trace("storing block in db")
	if err := c.loader.Append(&blk); err != nil {
		l.WithError(err).Error("block storing failed")
		return err
	}

	if err := c.db.Update(func(t database.Transaction) error {
		return t.ClearCandidateMessages()
	}); err != nil {
		l.WithError(err).Error("candidate deletion failed")
		return err
	}

	// 6. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// mempool.Mempool
	l.Trace("notifying internally")

	msg := message.New(topics.AcceptedBlock, blk)
	errList := c.eventBus.Publish(topics.AcceptedBlock, msg)
	diagnostics.LogPublishErrors("chain/chain.go, topics.AcceptedBlock", errList)

	l.Trace("procedure ended")
	return nil
}

// VerifyCandidateBlock can be used as a callback for the consensus in order to
// verify potential winning candidates.
func (c *Chain) VerifyCandidateBlock(blk block.Block) error {
	// We first perform a quick check on the Block Header and
	if err := c.verifier.SanityCheckBlock(*c.tip, blk); err != nil {
		return err
	}

	// TODO: consider using the context for timeouts
	_, err := c.proxy.Executor().VerifyStateTransition(c.ctx, blk.Txs, blk.Header.Height)
	return err
}

// Send Inventory message to all peers
func (c *Chain) advertiseBlock(b block.Block) error {
	// Disable gossiping messages if kadcast mode
	if config.Get().Kadcast.Enabled {
		return nil
	}

	msg := &message.Inv{}
	msg.AddItem(message.InvTypeBlock, b.Header.Hash)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		//TODO: shall this really panic ?
		log.Panic(err)
	}

	if err := topics.Prepend(buf, topics.Inv); err != nil {
		//TODO: shall this really panic ?
		log.Panic(err)
	}

	m := message.New(topics.Inv, *buf)
	errList := c.eventBus.Publish(topics.Gossip, m)
	diagnostics.LogPublishErrors("chain/chain.go, topics.Gossip, topics.Inv", errList)

	return nil
}

//nolint:unused
func (c *Chain) kadcastBlock(m message.Message) error {
	var kadHeight byte = 255
	if len(m.Header()) > 0 {
		kadHeight = m.Header()[0]
	}

	b, ok := m.Payload().(block.Block)
	if !ok {
		return errors.New("message payload not a block")
	}

	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, &b); err != nil {
		return err
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		return err
	}

	m = message.NewWithHeader(topics.Block, *buf, []byte{kadHeight})
	c.eventBus.Publish(topics.Kadcast, m)

	return nil
}

func (c *Chain) getRoundUpdate() consensus.RoundUpdate {
	return consensus.RoundUpdate{
		Round:           c.tip.Header.Height + 1,
		P:               c.p.Copy(),
		Seed:            c.tip.Header.Seed,
		Hash:            c.tip.Header.Hash,
		LastCertificate: c.tip.Header.Certificate,
	}
}

// GetSyncProgress returns how close the node is to being synced to the tip,
// as a percentage value.
// NOTE: this is just here to satisfy the grpc interface. It should be removed
// and the method should be moved to a synchronizer service.
func (c *Chain) GetSyncProgress(_ context.Context, e *node.EmptyRequest) (*node.SyncProgressResponse, error) {
	return &node.SyncProgressResponse{Progress: float32(100.0)}, nil
}

// RebuildChain will delete all blocks except for the genesis block,
// to allow for a full re-sync.
// NOTE: This function no longer does anything, but is still here to conform to the
// ChainServer interface, for GRPC communications.
func (c *Chain) RebuildChain(_ context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	return &node.GenericResponse{Response: "Unimplemented"}, nil
}

func (c *Chain) storeStakesInStormDB(blkHeight uint64) {
	store := capi.GetStormDBInstance()
	var members []*capi.Member
	for _, v := range c.p.Members {
		var stakes []capi.Stake

		for _, s := range v.Stakes {
			stake := capi.Stake{
				Amount:      s.Amount,
				StartHeight: s.StartHeight,
				EndHeight:   s.EndHeight,
			}
			stakes = append(stakes, stake)
		}

		member := capi.Member{
			PublicKeyBLS: v.PublicKeyBLS,
			Stakes:       stakes,
		}

		members = append(members, &member)
	}

	provisioner := capi.ProvisionerJSON{
		ID:      blkHeight,
		Set:     c.p.Set,
		Members: members,
	}
	err := store.Save(&provisioner)
	if err != nil {
		log.Warn("Could not store provisioners on memoryDB")
	}
}
