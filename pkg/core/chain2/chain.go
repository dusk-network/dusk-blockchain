package chain2

import (
	"context"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithFields(logrus.Fields{"process": "chain2"})

type signalType int

const (
	start signalType = 1
	quit  signalType = 0
)

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

	DB() database.DB
}

// Chain: Make DBLoader database.Driver-agnostic
//nolint:unused
type Chain struct {

	// Blockchain and consensus state
	loader Loader
	reg    *SafeRegistry

	// Blockchain and consensus state
	// loop    *loop.Consensus
	timeOut time.Duration

	// major set of components around consensus
	acceptor *Acceptor
	broker   *SafeRegistryBroker

	// chan to trigger consensus if already stopped
	QuitChan          chan bool
	accBlockChan      chan message.Message
	initConsensusChan chan message.Message

	// context to control child goroutines
	prntCtx    context.Context
	prntCancel context.CancelFunc

	// Node components
	eventBus eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	proxy    transactions.Proxy
	executor transactions.Executor
	ctx      context.Context
}

//nolint:unused
func New(ctx context.Context, e *eventbus.EventBus, r *rpcbus.RPCBus, proxy transactions.Proxy, loader Loader, executor transactions.Executor) (*Chain, error) {

	// Retrieve chain tip
	chainTip, err := loader.LoadTip()
	if err != nil {
		return nil, err
	}

	provisioners := user.NewProvisioners()
	var lastCertificate *block.Certificate

	if chainTip.Header.Height == 0 {
		// TODO: maybe it would be better to have a consensus-compatible certificate.
		lastCertificate = block.EmptyCertificate()

		// If we're running the test harness, we should also populate some consensus values
		if config.Get().Genesis.Legacy {
			if errV := setupBidValues(); errV != nil {
				return nil, errV
			}

			if errV := reconstructCommittee(provisioners, chainTip); errV != nil {
				return nil, errV
			}
		}
	}

	reg := newSafeRegistry(chainTip, lastCertificate, provisioners)

	// Acceptor instance
	acceptor, err := newAcceptor(*e, r, loader.DB(), executor, reg)
	if err != nil {
		return nil, err
	}

	// Acceptor instance
	broker, err := newSafeRegistryBroker(r, reg)
	if err != nil {
		return nil, err
	}

	accBlockChan := make(chan message.Message, 500)
	chanListener := eventbus.NewSafeChanListener(accBlockChan)
	e.Subscribe(topics.AcceptedBlock, chanListener)

	initConsensusChan := make(chan message.Message, 1)
	chanListener = eventbus.NewSafeChanListener(initConsensusChan)
	e.Subscribe(topics.Initialization, chanListener)

	// Parent context
	ctx, cancel := context.WithCancel(context.Background())
	// accepting blocks in the blockchain, alters SafeRegistry
	go acceptor.loop(ctx)

	// Provides async access (read/write) to SafeRegistry
	go broker.loop(ctx)

	return &Chain{
		eventBus:          *e,
		rpcBus:            r,
		loader:            loader,
		executor:          executor,
		proxy:             proxy,
		reg:               reg,
		broker:            broker,
		acceptor:          acceptor,
		prntCtx:           ctx,
		prntCancel:        cancel,
		accBlockChan:      accBlockChan,
		initConsensusChan: initConsensusChan,
		QuitChan:          make(chan bool),
		ctx:               ctx,
	}, nil
}

// ConsensusLoop
func (c *Chain) ConsensusLoop() {

	// Link a node instance to a wallet instance
	msg := <-c.initConsensusChan

	wk := msg.Payload().(key.WalletKeys)
	consensusKeys := wk.ConsensusKeys
	pubKey := wk.PublicKey

	// TODO: Chain context
	// TODO: Get Provisioners from Rusk on first run
	// TODO: Ensure candidateBroker should be reset on acceptBlock
	// TODO: Notify Chain for the Spin termination
	// TODO: consensus timeout config

	e := consensus.Emitter{
		EventBus:    &c.eventBus,
		RPCBus:      c.rpcBus,
		Keys:        consensusKeys,
		Proxy:       c.proxy,
		TimerLength: 10 * time.Second,
	}

	l := loop.New(&e)

	for {
		b := c.reg.GetChainTip()
		p := c.reg.GetProvisioners()

		// TODO: is this the correct way to check for empty provisioners
		if len(p.Members) == 0 {
			// TODO: is the client thread-safe?
			var err error
			p, err = c.executor.GetProvisioners(c.ctx)
			if err != nil {
				log.WithError(err).Error("Error in getting provisioners")
			}
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Initialize roundUpdate
		ru := consensus.RoundUpdate{
			Round: b.Header.Height + 1,
			P:     p.Copy(),
			Hash:  b.Header.Hash,
			Seed:  b.Header.Seed,
		}

		var wg sync.WaitGroup
		wg.Add(1)
		// Trigger consensus
		go func() {
			defer wg.Done()
			// Consensus spin is started in a separate goroutine
			// For stopping it, use StopLoopChan
			scr, agr, err := loop.CreateStateMachine(&e, c.loader.DB(), 10*time.Second, pubKey.Copy())
			if err != nil {
				log.WithError(err).Error("Could not initialize consensus spin")
				return
			}

			err = l.Spin(ctx, scr, agr, ru)
			if err != nil {
				log.WithError(err).Error("Consensus problem")
				return
			}
		}()

		// Support start/stop consensus spin
		if signal := c.wait(cancel, &wg); signal == quit {
			break
		}
	}
}

// TODO: Use utility goroutine to wrap consensus start/stop
func (c *Chain) wait(cancel context.CancelFunc, wg *sync.WaitGroup) signalType {
	for {
		select {
		// TODO Timeout
		case <-c.accBlockChan:
			// cancel consensus spin, return start signal
			cancel()
			wg.Wait()
			return start
		case <-c.QuitChan:
			cancel()
			wg.Wait()
			return quit
		}
	}
}

// Close should terminate/close safely Chain-related goroutines and data storages
func (c *Chain) Close() {

	// TODO: Stop consensus

	// Terminate child goroutines
	c.prntCancel()

	// Close DB
	_ = c.loader.DB().Close()
}
