package selection_test

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

func TestSelection(t *testing.T) {
	hlp := selection.NewHelper(10)
	consensusTimeOut := 300 * time.Millisecond

	ttestCB := func(require *require.Assertions, p consensus.InternalPacket, _ *eventbus.GossipStreamer) {
		require.NotNil(p)
		messageScore := p.(message.Score)
		require.NotEmpty(messageScore.Score)
	}

	testPhase := consensus.NewTestPhase(t, ttestCB, nil)
	sel := selection.New(testPhase, blockgenerator.Mock(hlp.Emitter), hlp.Emitter, consensusTimeOut)
	selFn := sel.Fn(nil)

	msgChan := make(chan message.Message, 1)
	msgs := hlp.Spawn()
	go func() {
		for _, msg := range msgs {
			msgChan <- message.New(topics.Score, msg)
		}
	}()
	testCallback := selFn(context.Background(), consensus.NewQueue(), msgChan, hlp.RoundUpdate(), hlp.Step)
	_ = testCallback(context.Background(), nil, nil, hlp.RoundUpdate(), hlp.Step+1)
}
