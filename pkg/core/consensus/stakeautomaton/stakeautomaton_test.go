package stakeautomaton_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
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
	bus, rb, p, keys := setupMaintainerTest(t)
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

	// add ourselves to the provisioners and the bidlist,
	// so that the maintainer gets the proper ending heights.
	// Provisioners
	member := consensus.MockMember(keys)
	member.Stakes[0].EndHeight = 1000
	p.Set.Insert(member.PublicKeyBLS)
	p.Members[string(member.PublicKeyBLS)] = member
	// Then, send a round update to update the values on the maintainer
	ru = consensus.MockRoundUpdate(2, p)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	go catchStakeRequest(t, rb, c)

	time.Sleep(100 * time.Millisecond)

	// Send another round update that is within the 'offset', to trigger sending a new pair of txs
	ru = consensus.MockRoundUpdate(950, p)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// Ensure stake request is sent
	<-c
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {
	bus, rb, p, _ := setupMaintainerTest(t)
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

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, *rpcbus.RPCBus, *user.Provisioners, key.Keys) {
	bus := eventbus.New()
	rpcBus := rpcbus.New()

	keys, err := key.NewRandKeys()
	require.Nil(t, err)

	m := stakeautomaton.New(bus, rpcBus, keys.BLSPubKeyBytes, nil)
	_, err = m.AutomateConsensusTxs(context.Background(), &node.EmptyRequest{})
	require.Nil(t, err)

	// Mock provisioners, and insert our wallet values
	p, _ := consensus.MockProvisioners(10)

	return bus, rpcBus, p, keys
}

func catchStakeRequest(t *testing.T, rb *rpcbus.RPCBus, respChan chan struct{}) {
	c := make(chan rpcbus.Request, 1)
	require.Nil(t, rb.Register(topics.SendStakeTx, c))

	<-c
	respChan <- struct{}{}
	rb.Deregister(topics.SendStakeTx)
}
