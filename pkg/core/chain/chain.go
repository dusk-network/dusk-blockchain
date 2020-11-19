package chain

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"encoding/hex"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
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

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB

	highestSeen uint64
	syncing     bool
	syncTarget  uint64
	*sequencer

	// loader abstracts away the persistence aspect of Block operations
	loader Loader

	// verifier performs verifications on the block
	verifier Verifier

	lock sync.RWMutex
	// current blockchain tip of local state
	tip *block.Block

	// Current set of provisioners
	p *user.Provisioners

	lastCertificate *block.Certificate
	pubKey          *keys.PublicKey

	// Consensus context, used to cancel the loop.
	consensusCtx context.Context
	interrupt    context.CancelFunc

	// Consensus loop
	loop      *loop.Consensus
	requestor *candidate.Requestor

	// rusk client
	proxy transactions.Proxy

	ctx context.Context
}

// New returns a new chain object. It accepts the EventBus (for messages coming
// from (remote) consensus components, the RPCBus for dispatching synchronous
// data related to Certificates, Blocks, Rounds and progress.
func New(ctx context.Context, db database.DB, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, loader Loader, verifier Verifier, srv *grpc.Server, proxy transactions.Proxy, requestor *candidate.Requestor) (*Chain, error) {
	chain := &Chain{
		eventBus:  eventBus,
		rpcBus:    rpcBus,
		db:        db,
		sequencer: newSequencer(),
		loader:    loader,
		verifier:  verifier,
		proxy:     proxy,
		ctx:       ctx,
		requestor: requestor,
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
		// TODO: maybe it would be better to have a consensus-compatible certificate.
		chain.lastCertificate = block.EmptyCertificate()

		// If we're running the test harness, we should also populate some consensus values
		if config.Get().Genesis.Legacy {
			if errV := setupBidValues(); errV != nil {
				return nil, errV
			}

			if errV := reconstructCommittee(chain.p, prevBlock); errV != nil {
				return nil, errV
			}
		}
	}

	if srv != nil {
		node.RegisterChainServer(srv, chain)
	}

	return chain, nil
}

// SetupConsensus adds the missing fields on the Chain which need to be populated
// by the user. Once the fields are populated, consensus is started.
// NOTE: we need locking here so that there is no race condition between
// the synchronization routine trying to figure out if there is a loop
// available, and the setting of these fields.
func (c *Chain) SetupConsensus(pk keys.PublicKey, l *loop.Consensus) {
	c.lock.Lock()
	c.pubKey = &pk
	c.loop = l
	c.lock.Unlock()

	// If we are on genesis and our consensus is being set up, it means
	// the network has just bootstrapped. So, we need to start the consensus,
	// and set up the interrupt.
	if c.tip.Header.Height == 0 {
		go func() {
			if err := c.startConsensus(); err != nil {
				log.WithError(err).Error("consensus failure")
			}
		}()
	}
}

// ProcessBlock will handle blocks incoming from the network. It will allow
// the chain to enter sync mode if it detects that we are behind, which will
// cancel the running consensus loop and attempt to reach the new chain tip.
// Satisfied the peer.ProcessorFunc interface.
func (c *Chain) ProcessBlock(m message.Message) ([]bytes.Buffer, error) {
	blk := m.Payload().(block.Block)
	log.WithField("height", blk.Header.Height).Trace("received block")

	c.lock.Lock()
	// Is it worth looking at this?
	if blk.Header.Height <= c.tip.Header.Height {
		log.Debug("discarded block from the past")
		c.lock.Unlock()
		return nil, nil
	}

	if blk.Header.Height > c.highestSeen {
		c.highestSeen = blk.Header.Height
	}

	// If we are more than one block behind, stop the consensus
	log.Debug("topics.StopConsensus")
	// FIXME: this call should be blocking
	if c.interrupt != nil {
		c.interrupt()
	}

	// If this block is from far in the future, we should start syncing mode.
	if blk.Header.Height > c.tip.Header.Height+1 {
		c.sequencer.add(blk)

		if !c.syncing {
			msgGetBlocks := createGetBlocksMsg(c.tip.Header.Hash)
			buf, err := marshalGetBlocks(msgGetBlocks)
			if err != nil {
				log.WithError(err).Error("could not marshalGetBlocks")
				c.lock.Unlock()
				return nil, err
			}

			c.syncTarget = blk.Header.Height
			if c.syncTarget > c.tip.Header.Height+config.MaxInvBlocks {
				c.syncTarget = c.tip.Header.Height + config.MaxInvBlocks
			}

			c.syncing = true
			c.lock.Unlock()
			return []bytes.Buffer{*buf}, nil
		}

		c.lock.Unlock()
		return nil, nil
	}

	// Otherwise, put it into the acceptance pipeline.
	return nil, c.onAcceptBlock(blk)
}

func createGetBlocksMsg(latestHash []byte) *message.GetBlocks {
	msg := &message.GetBlocks{}
	msg.Locators = append(msg.Locators, latestHash)
	return msg
}

//nolint:unparam
func marshalGetBlocks(msg *message.GetBlocks) (*bytes.Buffer, error) {
	buf := topics.GetBlocks.ToBuffer()
	if err := msg.Encode(&buf); err != nil {
		//FIXME: shall this panic here ?  result 1 (error) is always nil (unparam)
		//log.Panic(err)
		return nil, err
	}

	return &buf, nil
}

func (c *Chain) onAcceptBlock(blk block.Block) error {
	field := logger.Fields{"process": "onAcceptBlock", "height": blk.Header.Height}
	lg := log.WithFields(field)

	// Retrieve all successive blocks that need to be accepted
	blks := c.sequencer.provideSuccessors(blk)

	for _, blk := range blks {
		if err := c.AcceptBlock(c.ctx, blk); err != nil {
			lg.WithError(err).Debug("could not AcceptBlock")
			c.lock.Unlock()
			return err
		}
		c.lastCertificate = blk.Header.Certificate
	}

	// If we are no longer syncing after accepting this block,
	// request a certificate for the second to last round.
	if !c.syncing && c.pubKey != nil && c.loop != nil {
		// Once received, we can re-start consensus.
		// This sets off a chain of processing which goes from sending the
		// round update, to re-instantiating the consensus, to setting off
		// the first consensus loop. So, we do this in a goroutine to
		// avoid blocking other requests to the chain.
		c.lock.Unlock()
		return c.startConsensus()
	}

	c.lock.Unlock()
	return nil
}

// AcceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and stateful checks are true
// Returns nil, if checks passed and block was successfully saved
func (c *Chain) AcceptBlock(ctx context.Context, blk block.Block) error {
	field := logger.Fields{"process": "accept block", "height": blk.Header.Height}
	l := log.WithFields(field)

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

	provisioners, err := c.proxy.Executor().ExecuteStateTransition(ctx, blk.Txs, blk.Header.Height)
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

	// 5. Gossip advertise block Hash
	l.Trace("gossiping block")
	if err := c.advertiseBlock(blk); err != nil {
		l.WithError(err).Error("block advertising failed")
		return err
	}

	// 6. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// mempool.Mempool
	// consensus.generation.broker
	l.Trace("notifying internally")

	msg := message.New(topics.AcceptedBlock, blk)
	errList := c.eventBus.Publish(topics.AcceptedBlock, msg)
	diagnostics.LogPublishErrors("chain/chain.go, topics.AcceptedBlock", errList)

	if blk.Header.Height == c.syncTarget {
		l.Trace("ending sync")
		c.syncing = false
	}

	l.Trace("procedure ended")
	return nil
}

func (c *Chain) startConsensus() error {
	for {
		c.lock.Lock()
		ru := c.getRoundUpdate()
		c.consensusCtx, c.interrupt = context.WithCancel(c.ctx)
		scr, agr, err := loop.CreateStateMachine(c.loop.Emitter, c.db, config.ConsensusTimeOut, c.pubKey.Copy(), c.VerifyCandidateBlock, c.requestor)
		if err != nil {
			log.WithError(err).Error("could not create consensus state machine")
			c.lock.Unlock()
			return err
		}

		c.lock.Unlock()
		cert, blockHash, err := c.loop.Spin(c.consensusCtx, scr, agr, ru)
		if err != nil {
			// This is likely because of the consensus reaching max steps.
			// If this is the case, we simply propagate the error upwards.
			// TODO: maybe figure out a way to respond to this kind of error.
			return err
		}

		// If the context was canceled, the results will be nil. Thus, we can
		// break the loop and exit the function, to make place for the new one.
		if cert == nil || blockHash == nil {
			return nil
		}

		if err := c.handleCertificateMessage(cert, blockHash); err != nil {
			return err
		}
	}
}

// VerifyCandidateBlock can be used as a callback for the consensus in order to
// verify potential winning candidates.
func (c *Chain) VerifyCandidateBlock(blk block.Block) error {
	// We first perform a quick check on the Block Header and
	if err := c.verifier.SanityCheckBlock(*c.tip, blk); err != nil {
		return err
	}

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

func (c *Chain) handleCertificateMessage(cert *block.Certificate, blockHash []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lastCertificate = cert

	var cm block.Block
	if err := c.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(blockHash)
		return err
	}); err != nil {
		// If we can't get the block, we will fall
		// back and catch up later.
		//FIXME: restart consensus when handleCertificateMessage flow return err
		log.
			WithError(err).
			WithField("height", c.tip.Header.Height+1).
			Error("could not find winning candidate block")
		return err
	}

	// Try to accept candidate block
	cm.Header.Certificate = cert
	if err := c.AcceptBlock(c.ctx, cm); err != nil {
		log.
			WithError(err).
			WithField("candidate_hash", hex.EncodeToString(cm.Header.Hash)).
			WithField("candidate_height", cm.Header.Height).
			Error("could not accept candidate block")
		return err
	}

	return nil
}

func (c *Chain) getRoundUpdate() consensus.RoundUpdate {
	return consensus.RoundUpdate{
		Round:           c.tip.Header.Height + 1,
		P:               c.p.Copy(),
		Seed:            c.tip.Header.Seed,
		Hash:            c.tip.Header.Hash,
		LastCertificate: c.lastCertificate,
	}
}

// GetSyncProgress returns how close the node is to being synced to the tip,
// as a percentage value.
// TODO: fix
func (c *Chain) GetSyncProgress(ctx context.Context, e *node.EmptyRequest) (*node.SyncProgressResponse, error) {
	if c.highestSeen == 0 {
		return &node.SyncProgressResponse{Progress: 0}, nil
	}

	prevBlockHeight := c.tip.Header.Height
	progressPercentage := (float64(prevBlockHeight) / float64(c.highestSeen)) * 100

	// Avoiding strange output when the chain can be ahead of the highest
	// seen block, as in most cases, consensus terminates before we see
	// the new block from other peers.
	if progressPercentage > 100 {
		progressPercentage = 100
	}

	return &node.SyncProgressResponse{Progress: float32(progressPercentage)}, nil
}

// RebuildChain will delete all blocks except for the genesis block,
// to allow for a full re-sync.
// NOTE: This function no longer does anything, but is still here to conform to the
// ChainServer interface, for GRPC communications.
func (c *Chain) RebuildChain(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
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
