package testing

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

type signalType int

const (
	start signalType = 1
	quit  signalType = 0
)

//TODO: n.reg.ResetCandidates(round)

type mockChain struct {

	// Blockchain and consensus state
	db  database.DB
	reg *mockSafeRegistry

	// Blockchain and consensus state
	loop    *loop.Consensus
	timeOut time.Duration
	pubKey  *keys.PublicKey

	// major set of components around consensus
	acceptor *mockAcceptor
	broker   *mockSafeRegistryBroker

	// chan to trigger consensus if already stopped
	StartLoopChan, StopLoopChan chan bool
	QuitChan                    chan bool

	// context to control child goroutines
	prntCtx    context.Context
	prntCancel context.CancelFunc
}

func newMockChain(e consensus.Emitter, consensusTimeOut time.Duration, pubKey *keys.PublicKey, assert *assert.Assertions) (*mockChain, error) {

	// Open database driver
	drvr, err := database.From(lite.DriverName)
	if err != nil {
		return nil, err
	}

	db, err := drvr.Open("", protocol.TestNet, false)
	if err != nil {
		return nil, err
	}

	reg := newMockSafeRegistry()

	// Acceptor instance
	acceptor, err := newMockAcceptor(e, db, reg)
	assert.NoError(err)

	// Acceptor instance
	broker, err := newMockSafeRegistryBroker(e, reg)
	assert.NoError(err)

	loop := loop.New(&e)

	// Parent context
	ctx, cancel := context.WithCancel(context.Background())

	return &mockChain{
		db:            db,
		reg:           reg,
		loop:          loop,
		pubKey:        pubKey,
		timeOut:       consensusTimeOut,
		broker:        broker,
		acceptor:      acceptor,
		prntCtx:       ctx,
		prntCancel:    cancel,
		StartLoopChan: make(chan bool),
		StopLoopChan:  make(chan bool),
		QuitChan:      make(chan bool),
	}, nil
}

func (c *mockChain) MainLoop(p *user.Provisioners, assert *assert.Assertions) {

	// accepting blocks in the blockchain, alters SafeRegistry
	go c.acceptor.loop(c.prntCtx, assert)

	// Provides async access (read/write) to SafeRegistry
	go c.broker.loop(c.prntCtx, assert)

	// Chain main loop
	for {
		ctx, cancel := context.WithCancel(context.Background())

		b := c.reg.GetChainTip()
		lastRound := b.Header.Height

		// Initialize roundUpdate
		round := lastRound + 1
		hash, _ := crypto.RandEntropy(32)
		seed, _ := crypto.RandEntropy(32)
		ru := consensus.RoundUpdate{
			Round: round,
			P:     *p,
			Hash:  hash,
			Seed:  seed,
		}

		// Trigger consensus
		go func() {
			// Consensus spin is started in a separate goroutine
			// For stopping it, use StopLoopChan
			scr, agr, err := loop.CreateStateMachine(c.loop.Emitter, c.db, c.timeOut, c.pubKey.Copy())
			assert.NoError(err)

			err = c.loop.Spin(ctx, scr, agr, ru)
			assert.NoError(err)
		}()

		// Support start/stop consensus spin
		if s := c.WaitForSignal(cancel); s == quit {
			break
		}
	}

	c.teardown()
}

func (c *mockChain) WaitForSignal(cancel context.CancelFunc) signalType {
	for {
		select {
		case <-c.StopLoopChan:
			// cancel consensus spin and continue waiting for Start or Quit signals
			cancel()
		case <-c.StartLoopChan:
			// cancel consensus spin, return start signal
			cancel()
			return start
		case <-c.QuitChan:
			cancel()
			return quit
		}
	}
}

// teardown should terminate/close safely Chain-related goroutines and data storages
func (c *mockChain) teardown() {

	// Terminate child goroutines
	c.prntCancel()

	// TODO: Close DB
}
