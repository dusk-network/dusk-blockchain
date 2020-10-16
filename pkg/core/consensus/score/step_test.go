package score

import (
	"context"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/candidate"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/stretchr/testify/require"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func testResultFactory(require *require.Assertions, _ consensus.InternalPacket, streamer *eventbus.GossipStreamer) {
	_, err := streamer.Read()
	require.NoError(err)

	for i := 0; ; i++ {
		if i == 5 {
			break
		}
		tpcs := streamer.SeenTopics()
		for _, tpc := range tpcs {
			if tpc == topics.Score {
				return
			}
		}
		streamer.Read()
	}

	require.FailNow("no agreement received")
}

// TestSelectorRun tests that we can run the score
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
	hlp := NewHelper(50, time.Second)

	cb := func(ctx context.Context) bool {
		return true
	}

	d, _ := crypto.RandEntropy(32)
	k, _ := crypto.RandEntropy(32)
	edPk, _ := crypto.RandEntropy(32)

	mockPhase := consensus.MockPhase(cb)
	_, pk := transactions.MockKeys()

	blockGen := candidate.New(hlp.Emitter, pk)

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

	phaseFn := scoreInstance.Run(ctx, nil, nil, consensus.RoundUpdate{Round: round}, step)
	require.NotNil(t, phaseFn)

	_ = phaseFn(ctx, nil, nil, consensus.RoundUpdate{Round: round}, step+1)
	require.Nil(t, err)

	hlp.EventBus.Unsubscribe(topics.Gossip, id)

}
