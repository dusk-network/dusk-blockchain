package testing

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
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

	return &mockChain{
		db:       db,
		reg:      reg,
		loop:     loop,
		pubKey:   pubKey,
		timeOut:  consensusTimeOut,
		broker:   broker,
		acceptor: acceptor,
	}, nil
}

func (n *mockChain) Loop(assert *assert.Assertions) {

	// accepting blocks in the blockchain, alters SafeRegistry
	go n.acceptor.loop(assert)

	// Provides async access (read/write) to SafeRegistry
	go n.broker.loop(assert)

	// Chain main loop
	for {

		ctx, cancel := context.WithCancel(context.Background())

		b := n.reg.GetChainTip()
		lastRound := b.Header.Height

		// Initialize roundUpdate
		round := lastRound + 1
		ru := consensus.RoundUpdate{Round: uint64(round)}

		go func() {
			scr, agr, err := loop.CreateStateMachine(n.loop.Emitter, n.timeOut, n.pubKey.Copy())
			assert.NoError(err)

			err = n.loop.Spin(ctx, scr, agr, ru)
			assert.NoError(err)
		}()

		// Support start/stop consensus spin
		if s := n.WaitForSignal(cancel); s == quit {
			break
		}
	}
}

func (n *mockChain) WaitForSignal(cancel context.CancelFunc) signalType {
	for {
		select {
		case <-n.QuitChan:
			cancel()
			return quit
		case <-n.StopLoopChan:
			// cancel consensus spin and continue waiting for Start or Quit signals
			cancel()
		case <-n.StartLoopChan:
			cancel()
			return start
		}
	}
}
