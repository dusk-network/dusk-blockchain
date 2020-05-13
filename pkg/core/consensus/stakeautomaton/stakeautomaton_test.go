package stakeautomaton_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/stakeautomaton"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/stretchr/testify/require"
)

const pass = "password"

// Test that the maintainer will properly send new stake transactions, when
// one is about to expire, or if none exist.
func TestMaintainStakes(t *testing.T) {
	bus, rb, p := setupMaintainerTest(t)
	defer func() {
		_ = os.Remove("wallet.dat")
		_ = os.RemoveAll("walletDB")
	}()

	c := make(chan struct{}, 1)
	go catchStakeRequest(t, rb, c)

	// Send round update, to start the maintainer.
	ru := consensus.MockRoundUpdate(1, p)
	ruMsg := message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// Ensure stake request is sent
	<-c

	// Then, send a round update close after. No stake request should be sent
	ru = consensus.MockRoundUpdate(2, p)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	go catchStakeRequest(t, rb, c)

	select {
	case <-c:
		t.Fatal("was not supposed to get a tx in c")
	case <-time.After(1 * time.Second):
		// success
	}

	// Send another round update that is within the 'offset', to trigger sending a new pair of txs
	ru = consensus.MockRoundUpdate(950, p)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// Ensure stake request is sent
	<-c
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {
	bus, rb, p := setupMaintainerTest(t)
	defer func() {
		_ = os.Remove("wallet.dat")
		_ = os.RemoveAll("walletDB")
	}()

	c := make(chan struct{}, 1)
	go catchStakeRequest(t, rb, c)

	time.Sleep(100 * time.Millisecond)

	// Send round update, to start the maintainer.
	ru := consensus.MockRoundUpdate(1, p)
	ruMsg := message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// Ensure stake request is sent
	<-c

	go catchStakeRequest(t, rb, c)

	time.Sleep(100 * time.Millisecond)

	// Update round
	ru = consensus.MockRoundUpdate(2, p)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	select {
	case <-c:
		t.Fatal("was not supposed to get another tx in txChan")
	case <-time.After(2 * time.Second):
		// success
	}
}

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, *rpcbus.RPCBus, *user.Provisioners) {
	bus := eventbus.New()
	rpcBus := rpcbus.New()

	m := stakeautomaton.New(bus, rpcBus, nil)
	_, err := m.AutomateConsensusTxs(context.Background(), &node.EmptyRequest{})
	require.Nil(t, err)

	// Mock provisioners, and insert our wallet values
	p, _ := consensus.MockProvisioners(10)

	return bus, rpcBus, p
}

func catchStakeRequest(t *testing.T, rb *rpcbus.RPCBus, respChan chan struct{}) {
	c := make(chan rpcbus.Request, 1)
	require.Nil(t, rb.Register(topics.SendStakeTx, c))

	<-c
	respChan <- struct{}{}
	rb.Deregister(topics.SendStakeTx)
}
