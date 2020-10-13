package score

import (
	"context"
	"fmt"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/candidate"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/stretchr/testify/require"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func testResultFactory(require *require.Assertions, _ consensus.InternalPacket, streamer *eventbus.GossipStreamer) error {
	_, err := streamer.Read()
	require.NoError(err)

	for i := 0; ; i++ {
		if i == 2 {
			break
		}
		tpcs := streamer.SeenTopics()
		for _, tpc := range tpcs {
			if tpc == topics.Agreement {
				return nil
			}
		}
		streamer.Read()
	}

	return fmt.Errorf("no agreement received")
}

// TestSelectorRun tests that we can Run the selection
func TestScoreStepRun(t *testing.T) {
	round := uint64(1)
	step := uint8(1)

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	// creating the Helper
	hlp := reduction.NewHelper(50, time.Second)

	cb := func(ctx context.Context) (bool, error) {
		return true, nil
	}

	d, _ := crypto.RandEntropy(32)
	k, _ := crypto.RandEntropy(32)
	edPk, _ := crypto.RandEntropy(32)

	mockPhase := consensus.MockPhase(cb)
	blockGen := candidate.New(hlp.Emitter, &transactions.PublicKey{})

	scoreInstance := Phase{
		Emitter:   hlp.Emitter,
		bg:        hlp.Emitter.Proxy.BlockGenerator(),
		d:         d,
		k:         k,
		edPk:      edPk,
		threshold: consensus.NewThreshold(),
		next:      mockPhase,
		generator: blockGen,
	}

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	streamListener := eventbus.NewStreamListener(streamer)
	// wiring the Gossip streamer to capture the gossiped messages
	id := hlp.EventBus.Subscribe(topics.Gossip, streamListener)

	testPhase := consensus.NewTestPhase(t, testResultFactory, streamer)
	scoreInstance.SetNext(testPhase)

	ctx, canc := context.WithTimeout(context.Background(), 2*time.Second)
	defer canc()

	phaseFn, err := scoreInstance.Run(ctx, nil, nil, consensus.RoundUpdate{Round: round}, step)
	require.Nil(t, err)
	require.NotNil(t, phaseFn)

	_, err = phaseFn(ctx, nil, nil, consensus.RoundUpdate{Round: round}, step+1)
	require.Nil(t, err)

	hlp.EventBus.Unsubscribe(topics.Gossip, id)

}
