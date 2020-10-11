package selection_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

func TestSelection(t *testing.T) {
	consensusTimeOut := 300 * time.Millisecond

	ttestCB := func(require *require.Assertions, p consensus.InternalPacket, _ *eventbus.GossipStreamer) error {
		require.NotNil(p)

		if messageScore, ok := p.(message.Score); ok {
			require.NotEmpty(messageScore.Score)
			return nil
		}

		return errors.New("cb: failed to validate Score")
	}

	require := require.New(t)
	hlp := selection.NewHelper(10)
	testPhase := consensus.NewTestPhase(t, ttestCB, nil)
	sel := selection.New(testPhase, hlp.Emitter, consensusTimeOut)
	selFn := sel.Fn(nil)

	msgChan := make(chan message.Message, 1)
	msgs := hlp.Spawn()
	go func() {
		for _, msg := range msgs {
			msgChan <- message.New(topics.Score, msg)
		}
	}()
	testCallback, err := selFn(context.Background(), consensus.NewQueue(), msgChan, hlp.RoundUpdate(), hlp.Step)
	require.NoError(err)

	_, err = testCallback(context.Background(), nil, nil, hlp.RoundUpdate(), hlp.Step+1)
	require.NoError(err)
}
