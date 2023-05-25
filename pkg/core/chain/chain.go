// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/base58"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	errInvalidStateHash    = errors.New("invalid state hash")
	errUnexpectedStateHash = errors.New("unexpected state hash")

	log = logger.WithFields(logger.Fields{"process": "chain"})
)

// ErrBlockAlreadyAccepted block already known by blockchain state.
var ErrBlockAlreadyAccepted = errors.New("already accepted")

// TODO: This Verifier/Loader interface needs to be re-evaluated and most likely
// renamed. They don't make too much sense on their own (the `Loader` also
// appends blocks, and allows for fetching data from the DB), and potentially
// cause some clutter in the structure of the `Chain`.

// Verifier performs checks on the blockchain and potentially new incoming block.
type Verifier interface {
	// SanityCheckBlockchain on first N blocks and M last blocks.
	SanityCheckBlockchain(startAt uint64, firstBlocksAmount uint64) error
	// SanityCheckBlock will verify whether a block is valid according to the rules of the consensus.
	SanityCheckBlock(prevBlock block.Block, blk block.Block) error
}

// Loader is an interface which abstracts away the storage used by the Chain to
// store the blockchain.
type Loader interface {
	// LoadTip of the chain. Returns blockchain tip and persisted hash.
	LoadTip() (*block.Block, []byte, error)
	// Clear removes everything from the DB.
	Clear() error
	// Close the Loader and finalizes any pending connection.
	Close(driver string) error
	// Height returns the current height as stored in the loader.
	Height() (uint64, error)
	// BlockAt returns the block at a given height.
	BlockAt(uint64) (block.Block, error)
}

// Chain represents the nodes blockchain.
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB

	// loader abstracts away the persistence aspect of Block operations.
	loader Loader

	// verifier performs verifications on the block.
	verifier Verifier

	// current blockchain tip of local state.
	lock sync.RWMutex
	tip  *block.Block

	// Current set of provisioners.
	p *user.Provisioners

	// Consensus loop.
	loop              *loop.Consensus
	stopConsensusChan chan struct{}
	loopID            uint64

	// Syncing related things.
	*synchronizer
	highestSeen uint64

	// rusk client.
	proxy transactions.Proxy

	ctx context.Context

	blacklisted dupemap.TmpMap
	verified    sortedset.SafeSet
}

// New returns a new chain object. It accepts the EventBus (for messages coming
// from (remote) consensus components.
func New(ctx context.Context, db database.DB, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus,
	loader Loader, verifier Verifier, srv *grpc.Server, proxy transactions.Proxy, loop *loop.Consensus,
) (*Chain, error) {
	chain := &Chain{
		eventBus:          eventBus,
		rpcBus:            rpcBus,
		db:                db,
		loader:            loader,
		verifier:          verifier,
		proxy:             proxy,
		ctx:               ctx,
		loop:              loop,
		stopConsensusChan: make(chan struct{}),
		blacklisted:       *dupemap.NewTmpMap(1000, 120),
		verified:          sortedset.NewSafeSet(),
	}

	chain.synchronizer = newSynchronizer(db, chain)

	provisioners, err := proxy.Executor().GetProvisioners(ctx)
	if err != nil {
		log.WithError(err).Error("Error in getting provisioners")
		return nil, err
	}

	if srv != nil {
		node.RegisterChainServer(srv, chain)
	}

	chain.p = &provisioners

	if err := chain.syncWithRusk(); err != nil {
		return nil, err
	}

	return chain, nil
}

func (c *Chain) syncWithRusk() error {
	var (
		err           error
		ruskStateHash []byte
		persistedHash []byte
		prevBlock     *block.Block
	)

	ruskStateHash, err = c.proxy.Executor().GetStateRoot(c.ctx)
	if err != nil {
		return err
	}

	prevBlock, persistedHash, err = c.loader.LoadTip()
	if err != nil {
		return err
	}

	// Detect if both services are on the different state
	var persistedBlock *block.Block

	err = c.db.View(func(t database.Transaction) error {
		persistedBlock, err = t.FetchBlock(persistedHash)
		if err != nil {
			return err
		}

		if persistedBlock.Header.Height > 0 {
			if !bytes.Equal(persistedBlock.Header.StateHash, ruskStateHash) {
				log.WithField("rusk", hex.EncodeToString(ruskStateHash)).
					WithField("node", hex.EncodeToString(persistedBlock.Header.StateHash)).
					Error("invalid state detected")
				return errors.New("invalid state detected")
			}
		}

		return err
	})
	if err != nil {
		return err
	}

	// Update blockchain tip (in-memory)
	c.tip = persistedBlock

	// If both persisted block hash and latest blockchain block hash are the
	// same then there is no need to execute sync-up.
	if bytes.Equal(persistedHash, prevBlock.Header.Hash) {
		return nil
	}

	// re-accept missing block in order to recover Rusk (unpersisted) state.
	i := persistedBlock.Header.Height

	for {
		i++

		var blk *block.Block

		err = c.db.View(func(t database.Transaction) error {
			var hash []byte

			hash, err = t.FetchBlockHashByHeight(i)
			if err != nil {
				return err
			}

			blk, err = t.FetchBlock(hash)
			return err
		})

		if err != nil {
			break
		}

		// Re-accepting all blocks that have not been persisted in Rusk.
		// This will re-execute accept/finalize accordingly and update chain tip.
		if err := c.acceptBlock(*blk, false); err != nil {
			return err
		}
	}

	return nil
}

// ProcessBlockFromNetwork will handle blocks incoming from the network.
// It will allow the chain to enter sync mode if it detects that we are behind,
// which will cancel the running consensus loop and attempt to reach the new
// chain tip.
// Satisfies the peer.ProcessorFunc interface.
func (c *Chain) ProcessBlockFromNetwork(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	blk := m.Payload().(block.Block)

	// Ensure the received block provides a valid hash
	if err := verifiers.CheckHash(&blk); err != nil {
		return nil, err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	l := log.WithField("recv_blk_h", blk.Header.Height).
		WithField("curr_h", c.tip.Header.Height)

	if m.Metadata() != nil {
		l = l.WithField("kad_h", m.Metadata().KadcastHeight)
	}

	l.Trace("block received")

	h := blk.Header.Hash

	if c.blacklisted.Has(bytes.NewBuffer(h)) {
		log.WithField("hash", util.StringifyBytes(h)).Warn("filter out blacklisted block")
		return nil, nil
	}

	switch {
	case blk.Header.Height == c.tip.Header.Height:
		{
			// Check if we already accepted this block
			if bytes.Equal(blk.Header.Hash, c.tip.Header.Hash) {
				l.WithError(ErrBlockAlreadyAccepted).Debug("discard block")
				return nil, nil
			}

			hash := c.tip.Header.Hash

			// Try to fallback
			if err := c.tryFallback(blk); err != nil {
				l.WithError(err).Error("failed fallback procedure")
				return nil, nil
			}

			// Fallback completed successfully. This means that the old tip hash
			// came from a consensus fork. That's said, we should filter it
			// out if any other node propagates it back when this node is syncing up.
			c.blacklisted.Add(bytes.NewBuffer(hash))

			return c.synchronizer.processBlock(srcPeerID, c.tip.Header.Height, blk, m.Metadata())
		}
	case blk.Header.Height < c.tip.Header.Height:
		l.Debug("discard block")

		// Due to a network glitch, the fallback procedure may be skipped.
		// In this case, network may continue on a branch with higher iteration value.
		// Here we try to detect the above edge case.
		if res, err := c.isBlockFromFork(blk); err != nil {
			l.WithError(err).Warn("invalid block")
		} else {
			if res {
				l.WithField("recv_blk_iteration", blk.Header.Iteration).
					WithField("recv_blk_hash", hex.EncodeToString(h)).
					WithField("event", "fallback").Error("fork detected")
			}
		}

		return nil, nil
	}

	if blk.Header.Height > c.highestSeen {
		c.highestSeen = blk.Header.Height
	}

	return c.synchronizer.processBlock(srcPeerID, c.tip.Header.Height, blk, m.Metadata())
}

// TryNextConsecutiveBlockOutSync is the processing path for accepting a block
// from the network during out-of-sync state.
func (c *Chain) TryNextConsecutiveBlockOutSync(blk block.Block, metadata *message.Metadata) error {
	log.WithField("height", blk.Header.Height).Trace("accepting sync block")
	return c.acceptBlock(blk, true)
}

// TryNextConsecutiveBlockInSync is the processing path for accepting a block
// from the network during in-sync state. Returns err if the block is not valid.
func (c *Chain) TryNextConsecutiveBlockInSync(blk block.Block, metadata *message.Metadata) error {
	// Make an attempt to accept a new block. If succeeds, we could safely restart the Consensus Loop.
	// If not, peer reputation score should be decreased.
	if err := c.acceptSuccessiveBlock(blk, metadata); err != nil {
		return err
	}

	// Consensus needs a fresh restart so that it is initialized with most
	// recent round update which is Chain tip and the list of active Provisioners.
	if err := c.RestartConsensus(); err != nil {
		log.WithError(err).Error("failed to start consensus loop")
	}

	return nil
}

// TryNextConsecutiveBlockIsValid makes an attempt to validate a blk without
// changing any state.
// returns error if the block is invalid to current blockchain tip.
func (c *Chain) TryNextConsecutiveBlockIsValid(blk block.Block) error {
	fields := logger.Fields{
		"event":    "check_block",
		"height":   blk.Header.Height,
		"hash":     util.StringifyBytes(blk.Header.Hash),
		"curr_h":   c.tip.Header.Height,
		"prov_num": c.p.Set.Len(),
	}

	l := log.WithFields(fields)

	return c.isValidHeader(blk, *c.tip, *c.p, l, true)
}

// ProcessSyncTimerExpired called by outsync timer when a peer does not provide GetData response.
// It implements transition back to inSync state.
// strPeerAddr is the address of the peer initiated the syncing but failed to deliver.
func (c *Chain) ProcessSyncTimerExpired(strPeerAddr string) error {
	log.WithField("curr", c.tip.Header.Height).
		WithField("src_addr", strPeerAddr).Warn("sync timer expired")

	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.RestartConsensus(); err != nil {
		log.WithError(err).Warn("sync timer could not restart consensus loop")
	}

	log.WithField("state", "inSync").Traceln("change sync state")

	c.state = c.inSync
	return nil
}

// acceptSuccessiveBlock will accept a block which directly follows the chain
// tip, and advertises it to the node's peers.
func (c *Chain) acceptSuccessiveBlock(blk block.Block, metadata *message.Metadata) error {
	log.WithField("height", blk.Header.Height).Trace("accepting succeeding block")

	if err := c.isValidHeader(blk, *c.tip, *c.p, log, true); err != nil {
		log.WithError(err).Error("invalid block")
		return err
	}

	if err := c.kadcastBlock(blk, metadata); err != nil {
		log.WithError(err).Error("block propagation failed")
		return err
	}

	if err := c.acceptBlock(blk, true); err != nil {
		return err
	}

	if blk.Header.Height > c.highestSeen {
		c.highestSeen = blk.Header.Height
	}

	return nil
}

// runStateTransition performs state transition and returns a block with gasSpent field populated for each tx.
func (c *Chain) runStateTransition(tipBlk, blk block.Block) (*block.Block, error) {
	var (
		respStateHash       []byte
		provisionersUpdated user.Provisioners
		err                 error
		provisionersCount   int

		fields = logger.Fields{
			"event":      "accept_block",
			"height":     blk.Header.Height,
			"iteration":  blk.Header.Iteration,
			"hash":       util.StringifyBytes(blk.Header.Hash),
			"curr_h":     c.tip.Header.Height,
			"block_time": blk.Header.Timestamp - tipBlk.Header.Timestamp,
			"txs_count":  len(blk.Txs),
		}

		l = log.WithFields(fields)
	)

	if err = c.sanityCheckStateHash(); err != nil {
		return block.NewBlock(), err
	}

	provisionersCount = c.p.Set.Len()
	eligibleProvisioners := c.p.SubsetSizeAt(c.tip.Header.Height)

	l.WithField("prov", provisionersCount).
		WithField("e_prov", eligibleProvisioners).
		Info("run state transition")

	var txs []transactions.ContractCall

	switch blk.Header.Iteration {
	case 1:
		// Finalized block. first iteration consensus agreement.
		txs, provisionersUpdated, respStateHash, err = c.proxy.Executor().Finalize(c.ctx,
			blk.Txs,
			tipBlk.Header.StateHash,
			blk.Header.Height,
			blk.Header.GasLimit,
			blk.Header.GeneratorBlsPubkey,
			c.p,
		)
		if err != nil {
			l.WithError(err).
				WithField("grpc", "finalize").
				Error("Error in executing the state transition")
			return block.NewBlock(), err
		}
	default:
		missedIterations := blk.Header.Iteration - 1
		for iteration := uint8(0); iteration < missedIterations; iteration++ {
			step := iteration*3 + 1
			committee := c.p.CreateVotingCommittee(tipBlk.Header.Seed, blk.Header.Height, step, config.ConsensusSelectionCommitteeSize)
			committeeKeys := committee.MemberKeys()

			if len(committeeKeys) == 1 {
				expectedkey, _ := base58.Encode(committeeKeys[0])
				log.
					WithField("iteration", iteration+1).
					WithField("height", blk.Header.Height).
					WithField("generator", expectedkey).
					Warn("Missed block from provisioner")
			} else {
				log.
					WithField("iteration", iteration+1).
					WithField("height", blk.Header.Height).
					Error("Unable to generate voting committee for missed block")
			}
		}

		// Tentative block. non-first iteration consensus agreement.
		txs, provisionersUpdated, respStateHash, err = c.proxy.Executor().Accept(c.ctx,
			blk.Txs,
			tipBlk.Header.StateHash,
			blk.Header.Height,
			blk.Header.GasLimit, blk.Header.GeneratorBlsPubkey, c.p)
		if err != nil {
			l.WithError(err).
				WithField("grpc", "accept").
				Error("Error in executing the state transition")

			return block.NewBlock(), err
		}
	}

	// Sanity check to ensure accepted block state_hash is the same as the one Finalize/Accept returned.
	if !bytes.Equal(respStateHash, blk.Header.StateHash) {
		log.WithField("rusk", util.StringifyBytes(respStateHash)).
			WithField("node", util.StringifyBytes(blk.Header.StateHash)).
			WithError(errInvalidStateHash).Error("inconsistency with state_hash")

		return block.NewBlock(), errInvalidStateHash
	}

	// Tamper block transactions with ones return by Rusk service in order to persist GasSpent per transaction.
	for _, tx := range txs {
		h, err := tx.CalculateHash()
		if err != nil {
			log.WithError(err).Warn("could not read rusk tx hash")
		}

		if tx.TxError() != nil {
			log.WithField("desc", tx.TxError().String()).Warn("transaction rusk error")
		}

		if err := blk.TamperExecutedTransaction(h, tx.GasSpent(), tx.TxError()); err != nil {
			log.WithError(err).Warn("could not tamper ExecutedTransaction")
		}
	}

	// Update the provisioners.
	// blk.Txs may bring new provisioners to the current state
	c.p = &provisionersUpdated
	eligibleProvisioners = c.p.SubsetSizeAt(c.tip.Header.Height + 1)

	l.WithField("prov", c.p.Set.Len()).
		WithField("added", c.p.Set.Len()-provisionersCount).
		WithField("state_hash", util.StringifyBytes(respStateHash)).WithField("e_prov", eligibleProvisioners).
		Info("state transition completed")

	provisioner, _ := base58.Encode(blk.Header.GeneratorBlsPubkey)
	logger.WithField("generator", provisioner).
		WithField("iteration", blk.Header.Iteration).
		WithField("height", blk.Header.Height).
		Info("Accepted block from provisioner")

	return &blk, nil
}

// sanityCheckStateHash ensures most recent local statehash and rusk statehash are the same.
func (c *Chain) sanityCheckStateHash() error {
	if c.tip.Header.Height == 0 {
		return nil
	}

	// Ensure that both (co-deployed) services node and rusk are on the same
	// state. If not, we should trigger a recovery procedure so both are
	// always synced up.
	ruskStateHash, err := c.proxy.Executor().GetStateRoot(c.ctx)
	if err != nil {
		return err
	}

	nodeStateHash := c.tip.Header.StateHash

	if !bytes.Equal(nodeStateHash, ruskStateHash) || len(nodeStateHash) == 0 {
		log.WithField("rusk", hex.EncodeToString(ruskStateHash)).
			WithError(errInvalidStateHash).
			WithField("node", hex.EncodeToString(nodeStateHash)).
			Error("check state_hash failed")

		return errInvalidStateHash
	}

	return nil
}

func (c *Chain) isValidHeader(newBlock, prevBlock block.Block, provisioners user.Provisioners, l *logrus.Entry, withSanityCheck bool) error {
	l.Debug("verifying block header")
	// Check that stateless and stateful checks pass
	if withSanityCheck {
		if err := c.verifier.SanityCheckBlock(prevBlock, newBlock); err != nil {
			l.WithError(err).Error("block header verification failed")
			return err
		}
	}

	// Check the certificate
	// This check should avoid a possible race condition between accepting two blocks
	// at the same height, as the probability of the committee creating two valid certificates
	// for the same round is negligible.
	l.Debug("verifying block certificate")

	var err error
	if err = agreement.CheckBlockCertificate(provisioners, newBlock, prevBlock.Header.Seed); err != nil {
		l.WithError(err).Error("certificate verification failed")
		return err
	}

	return nil
}

// acceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and stateful checks are true
// Returns nil, if checks passed and block was successfully saved.
func (c *Chain) acceptBlock(blk block.Block, withSanityCheck bool) error {
	fields := logger.Fields{
		"event":     "accept_block",
		"height":    blk.Header.Height,
		"iteration": blk.Header.Iteration,
		"hash":      util.StringifyBytes(blk.Header.Hash),
		"curr_h":    c.tip.Header.Height,
		"prov_num":  c.p.Set.Len(),
	}

	l := log.WithFields(fields)
	var err error

	// 1. Ensure block fields and certificate are valid
	if err = c.isValidHeader(blk, *c.tip, *c.p, l, withSanityCheck); err != nil {
		l.WithError(err).Error("invalid block error")
		return err
	}

	// 2. Perform State Transition to update Contract Storage with Tentative or Finalized state.
	var b *block.Block

	if b, err = c.runStateTransition(*c.tip, blk); err != nil {
		l.WithError(err).Error("execute state transition failed")
		return err
	}

	// 3. Persist the approved block and update in-memory chain tip
	l.Debug("persisting block")

	if err := c.persist(b); err != nil {
		l.WithError(err).Error("persisting block failed")
		return err
	}

	c.tip = b
	c.verified.Reset()

	// 5. Perform all post-events on accepting a block
	c.postAcceptBlock(*b, l)

	return nil
}

// Persist persists a block in both Contract Storage state and dusk-blockchain db in atomic manner.
func (c *Chain) persist(b *block.Block) error {
	var (
		clog = log.WithFields(logger.Fields{
			"event":  "accept_block",
			"height": b.Header.Height,
			"hash":   util.StringifyBytes(b.Header.Hash),
			"curr_h": c.tip.Header.Height,
		})

		err error
		pe  = config.Get().State.PersistEvery
	)

	//  Atomic persist
	err = c.db.Update(func(t database.Transaction) error {
		var p bool

		if pe > 0 && b.Header.Height%pe == 0 {
			// Mark it as a persisted block
			p = true
		}

		// Persist block into dusk-blockchain database before any attempt to persist in Rusk.
		// If StoreBlock fails, no change will be applied in Rusk.
		// If Rusk.Persist fails, StoreBlock is rollbacked.
		if err = t.StoreBlock(b, p); err != nil {
			return err
		}

		// Persist Rusk state
		if p {
			if err = c.proxy.Executor().Persist(c.ctx, b.Header.StateHash); err != nil {
				clog.WithError(err).Error("persisting contract state failed")
				return err
			}

			clog.Debug("persisting contract state completed")
		}

		return nil
	})

	return err
}

// postAcceptBlock performs all post-events on accepting a block.
func (c *Chain) postAcceptBlock(blk block.Block, l *logrus.Entry) {
	// 1. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// mempool.Mempool
	l.Debug("notifying internally")

	msg := message.New(topics.AcceptedBlock, blk)
	errList := c.eventBus.Publish(topics.AcceptedBlock, msg)

	// 2. Clear obsolete Candidate blocks
	if err := c.db.Update(func(t database.Transaction) error {
		return t.ClearCandidateMessages()
	}); err != nil {
		// failure here should not be treated as critical
		l.WithError(err).Warn("candidate deletion failed")
	}

	diagnostics.LogPublishErrors("chain/chain.go, topics.AcceptedBlock", errList)
	l.Debug("procedure ended")
}

// VerifyCandidateBlock can be used as a callback for the consensus in order to
// verify potential winning candidates.
func (c *Chain) VerifyCandidateBlock(ctx context.Context, candidate block.Block) error {
	var (
		err       error
		chainTip  block.Block
		stateRoot []byte
	)

	c.lock.Lock()
	chainTip = c.tip.Copy().(block.Block)
	c.lock.Unlock()

	// A edge case where the next valid block is received from the network while
	// consensus loop is still running over an old state.
	if chainTip.Header.Height >= candidate.Header.Height {
		return reduction.ErrLowBlockHeight
	}

	// We first perform a quick check on the Block Header
	err = c.verifier.SanityCheckBlock(chainTip, candidate)
	if err != nil {
		return err
	}

	// Locking here would enable Chain to perform VST calls in a row, checking
	// hash against cached hashes firstly.
	c.verified.Lock()
	defer c.verified.Unlock()

	if c.verified.Contains(candidate.Header.Hash) {
		// already verified
		return nil
	}

	stateRoot, err = c.proxy.Executor().VerifyStateTransition(ctx, candidate.Txs, candidate.Header.GasLimit,
		candidate.Header.Height, candidate.Header.GeneratorBlsPubkey)
	if err != nil {
		return err
	}

	c.verified.Insert(candidate.Header.Hash)

	if !bytes.Equal(stateRoot, candidate.Header.StateHash) {
		log.WithField("candidate_state_hash", hex.EncodeToString(candidate.Header.StateHash)).
			WithField("vst_state_hash", hex.EncodeToString(stateRoot)).Error(errUnexpectedStateHash.Error())

		return errUnexpectedStateHash
	}

	return nil
}

// ExecuteStateTransition calls Rusk ExecuteStateTransitiongrpc method.
func (c *Chain) ExecuteStateTransition(ctx context.Context, txs []transactions.ContractCall, blockHeight uint64, blockGasLimit uint64, generator []byte) ([]transactions.ContractCall, []byte, error) {
	return c.proxy.Executor().ExecuteStateTransition(c.ctx, txs, blockGasLimit, blockHeight, generator)
}

func (c *Chain) kadcastBlock(blk block.Block, metadata *message.Metadata) error {
	log.WithField("blk_height", blk.Header.Height).Trace("propagate block")

	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, &blk); err != nil {
		return err
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		return err
	}

	c.eventBus.Publish(topics.Kadcast, message.NewWithMetadata(topics.Block, *buf, metadata))
	return nil
}

// getRoundUpdate constructs RoundUpdate and returns a deep copy.
func (c *Chain) getRoundUpdate() consensus.RoundUpdate {
	r := consensus.RoundUpdate{
		Round:           c.tip.Header.Height + 1,
		P:               *c.p,
		Seed:            c.tip.Header.Seed,
		Hash:            c.tip.Header.Hash,
		LastCertificate: c.tip.Header.Certificate,
		Timestamp:       c.tip.Header.Timestamp,
	}

	return r.Copy().(consensus.RoundUpdate)
}

// GetSyncProgress returns how close the node is to being synced to the tip,
// as a percentage value.
func (c *Chain) GetSyncProgress(_ context.Context, e *node.EmptyRequest) (*node.SyncProgressResponse, error) {
	return &node.SyncProgressResponse{Progress: float32(c.CalculateSyncProgress())}, nil
}

// CalculateSyncProgress of the node.
func (c *Chain) CalculateSyncProgress() float64 {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.highestSeen == 0 {
		return 0.0
	}

	progressPercentage := (float64(c.tip.Header.Height) / float64(c.highestSeen)) * 100
	if progressPercentage > 100 {
		progressPercentage = 100
	}

	return progressPercentage
}

// RebuildChain will delete all blocks except for the genesis block,
// to allow for a full re-sync.
// NOTE: This function no longer does anything, but is still here to conform to the
// ChainServer interface, for GRPC communications.
func (c *Chain) RebuildChain(_ context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	return &node.GenericResponse{Response: "Unimplemented"}, nil
}

//nolint
func (c *Chain) storeStakesInStormDB(blkHeight uint64) {
	store := capi.GetStormDBInstance()
	members := make([]*capi.Member, len(c.p.Members))
	i := 0

	for _, v := range c.p.Members {
		var stakes []capi.Stake

		for _, s := range v.Stakes {
			stake := capi.Stake{
				Value:       s.Value,
				Reward:      s.Reward,
				Counter:     s.Counter,
				Eligibility: s.Eligibility,
			}

			stakes = append(stakes, stake)
		}

		member := capi.Member{
			PublicKeyBLS: v.PublicKeyBLS,
			Stakes:       stakes,
		}

		members[i] = &member
		i++
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
