package chain2

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Acceptor accepts block on topics.Certificate, topics.Block and provides VerifyCandidateBlock
// Acceptor owns mockConsensusRegistry and has direct read/write access to it
//nolint:unused
type Acceptor struct {
	certficateChan           chan message.Message
	blockChan                chan message.Message
	verifyCandidateBlockChan chan rpcbus.Request

	// reference to chain state (not owner)
	db  database.DB
	reg *SafeRegistry

	eventBus eventbus.EventBus
	rpcbus   *rpcbus.RPCBus

	// rusk client
	executor transactions.Executor
	ctx      context.Context
}

//nolint:unused
func newAcceptor(e eventbus.EventBus, r *rpcbus.RPCBus, db database.DB, executor transactions.Executor, reg *SafeRegistry) (*Acceptor, error) {

	// Subscriptions
	certficateChan := make(chan message.Message, 1)
	chanListener := eventbus.NewSafeChanListener(certficateChan)
	e.Subscribe(topics.Certificate, chanListener)

	blockChan := make(chan message.Message, 500)
	chanListener = eventbus.NewSafeChanListener(blockChan)
	e.Subscribe(topics.Block, chanListener)

	verifyCandidateBlockChan := make(chan rpcbus.Request, 10)
	if err := r.Register(topics.VerifyCandidateBlock, verifyCandidateBlockChan); err != nil {
		return nil, err
	}

	acc := Acceptor{
		certficateChan:           certficateChan,
		blockChan:                blockChan,
		verifyCandidateBlockChan: verifyCandidateBlockChan,
		db:                       db,
		reg:                      reg,
		eventBus:                 e,
		rpcbus:                   r,
		executor:                 executor,
	}

	return &acc, nil
}

// acceptBlock verifies block from consensus perspective
// Non-duplicated block
// Valid block certificate
// Valid block header
func (a *Acceptor) acceptBlock(b block.Block) error {

	// 1. Check the certificate
	if err := sanityCheckBlock(a.db, a.reg.GetChainTip(), b); err != nil {
		return err
	}

	// TODO: ctx
	ctx, _ := context.WithCancel(context.Background())

	var p user.Provisioners
	var err error
	p, err = a.executor.GetProvisioners(ctx)
	if err != nil {
		log.WithError(err).Error("Error in getting provisioners")
		return err
	}

	a.reg.SetProvisioners(p)

	// 2. Check the certificate
	// This check should avoid a possible race condition between accepting two blocks
	// at the same height, as the probability of the committee creating two valid certificates
	// for the same round is negligible.
	if err := verifiers.CheckBlockCertificate(a.reg.GetProvisioners(), b); err != nil {
		return err
	}

	// 3. Call ExecuteStateTransitionFunction
	{
		p, err := a.executor.ExecuteStateTransition(ctx, b.Txs, b.Header.Height)
		if err != nil {
			log.WithError(err).Error("Error in executing the state transition")
			return err
		}

		a.reg.SetProvisioners(p)
	}

	// Store block in the in-memory database
	err = a.db.Update(func(t database.Transaction) error {
		return t.StoreBlock(&b)
	})

	if err != nil {
		return err
	}

	// Update registry
	a.reg.SetChainTip(b)

	// Gossip/Kadcast here topics.Block
	if err := a.advertiseBlock(b); err != nil {
		return err
	}

	// 6. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// Mempool
	// Chain.ConsensusLoop
	// Candidate/Broker
	// Block Explorer
	msg := message.New(topics.AcceptedBlock, b)
	_ = a.eventBus.Publish(topics.AcceptedBlock, msg)

	return err
}

func (a *Acceptor) processCandidateVerificationRequest(r rpcbus.Request) {
	var res rpcbus.Response

	cm := r.Params.(message.Candidate)
	candidateBlock := *cm.Block
	chainTip := a.reg.GetChainTip()

	// We first perform a quick check on the Block Header and
	if err := sanityCheckBlock(a.db, chainTip, candidateBlock); err != nil {
		res.Err = err
		r.RespChan <- res
		return
	}
	// TODO: ctx
	ctx, _ := context.WithCancel(context.Background())
	_, err := a.executor.VerifyStateTransition(ctx, candidateBlock.Txs, candidateBlock.Header.Height)
	if err != nil {
		res.Err = err
		r.RespChan <- res
		return
	}

	r.RespChan <- res
}

func (a *Acceptor) loop(pctx context.Context) {

	for {
		select {
		// Handles topics.Certificate from consensus
		case m := <-a.certficateChan:

			// Extract winning hash and block certificate
			cMsg := m.Payload().(message.Certificate)
			certificate := cMsg.Ag.GenerateCertificate()
			winningHash := cMsg.Ag.State().BlockHash

			// Try to fetch block by hash from local state or network
			cm, err := GetCandidate(a.rpcbus, winningHash)
			if err != nil {
				log.Error("could not accept block")
				continue
			}
			cm.Block.Header.Certificate = certificate
			// Ensure block is accepted by Chain
			if err = a.acceptBlock(*cm.Block); err != nil {
				log.Error("could not accept block from consensus")
			}

			// Handles topics.Block from the wire on synchronizing
		case msg := <-a.blockChan:
			b := msg.Payload().Copy().(block.Block)
			if err := a.acceptBlock(b); err != nil {
				log.Error("could not accept block from network")
			}
		case req := <-a.verifyCandidateBlockChan:
			a.processCandidateVerificationRequest(req)
			// Handles Idle
		case <-time.After(30 * time.Second):
			log.Warn("acceptor on idle")
		case <-pctx.Done():
			return
		}
	}
}

func GetCandidate(r *rpcbus.RPCBus, hash []byte) (message.Candidate, error) {

	params := new(bytes.Buffer)
	_ = encoding.Write256(params, hash)
	_ = encoding.WriteBool(params, true)

	timeoutGetCandidate := time.Duration(config.Get().Timeout.TimeoutGetCandidate) * time.Second
	resp, err := r.Call(topics.GetCandidate, rpcbus.NewRequest(*params), timeoutGetCandidate) //20 is tmp value for further checks
	if err != nil {
		// If the we can't get the block, we will fall
		// back and catch up later.
		log.
			WithError(err).
			WithField("hash", hex.EncodeToString(hash)).
			Error("could not find candidate block")
		return message.Candidate{}, err
	}

	return resp.(message.Candidate), nil
}

func sanityCheckBlock(db database.DB, prevBlock block.Block, b block.Block) error {

	// TODO: use loader.sanityCheckBlock

	// 1. Check if the block is a duplicate
	err := db.View(func(t database.Transaction) error {
		_, err := t.FetchBlockExists(b.Header.Hash)
		return err
	})

	if err != database.ErrBlockNotFound {
		if err == nil {
			err = errors.New("block already exists")
		}
		return err
	}

	// SanityCheck block header to ensure consensus has worked a chainTip
	if err = verifiers.CheckBlockHeader(prevBlock, b); err != nil {
		return err
	}

	if err = verifiers.CheckMultiCoinbases(b.Txs); err != nil {
		return err
	}

	return nil
}

// Send Inventory message to all peers
func (a *Acceptor) advertiseBlock(b block.Block) error {
	msg := &peermsg.Inv{}
	msg.AddItem(peermsg.InvTypeBlock, b.Header.Hash)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		log.Panic(err)
	}

	if err := topics.Prepend(buf, topics.Inv); err != nil {
		log.Panic(err)
	}

	m := message.New(topics.Inv, *buf)
	errList := a.eventBus.Publish(topics.Gossip, m)
	diagnostics.LogPublishErrors("chain/chain.go, topics.Gossip, topics.Inv", errList)

	return nil
}
