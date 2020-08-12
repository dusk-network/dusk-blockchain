package bidautomaton_test

import (
	"context"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/bidautomaton"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/stretchr/testify/require"
)

// Test that the maintainer will properly send new bid transactions, when
// one is about to expire, or if none exist.
func TestMaintainBids(t *testing.T) {

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	bus, rb := setupMaintainerTest(t)
	defer func() {
		_ = os.Remove("wallet.dat")
		_ = os.RemoveAll("walletDB")
	}()

	c := make(chan struct{}, 1)
	catchStakeRequest(rb, c)

	// Send round update, to start the maintainer.
	ru := consensus.MockRoundUpdate(1, nil)
	ruMsg := message.New(topics.RoundUpdate, ru)
	errList := bus.Publish(topics.RoundUpdate, ruMsg)
	require.Empty(t, errList)

	// Ensure bid request is sent
	<-c

	// Then, send a round update close after. No bid request should be sent
	ru = consensus.MockRoundUpdate(2, nil)
	ruMsg = message.New(topics.RoundUpdate, ru)
	errList = bus.Publish(topics.RoundUpdate, ruMsg)
	require.Empty(t, errList)

	catchStakeRequest(rb, c)

	select {
	case <-c:
		t.Fatal("was not supposed to get a tx in c")
	case <-time.After(1 * time.Second):
		// success
	}

	// Send another round update that is within the 'offset', to trigger sending a new tx
	ru = consensus.MockRoundUpdate(950, nil)
	ruMsg = message.New(topics.RoundUpdate, ru)
	errList = bus.Publish(topics.RoundUpdate, ruMsg)
	require.Empty(t, errList)

	// Ensure bid request is sent
	<-c
}

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, *rpcbus.RPCBus) {
	bus := eventbus.New()
	rpcBus := rpcbus.New()

	m := bidautomaton.New(bus, rpcBus, nil)
	_, err := m.AutomateBids(context.Background(), &node.EmptyRequest{})
	require.Nil(t, err)

	return bus, rpcBus
}

func catchStakeRequest(rb *rpcbus.RPCBus, respChan chan struct{}) {
	c := make(chan rpcbus.Request, 1)
	if err := rb.Register(topics.SendBidTx, c); err != nil {
		panic(err)
	}

	go func() {
		<-c
		respChan <- struct{}{}
		rb.Deregister(topics.SendBidTx)
	}()
}
