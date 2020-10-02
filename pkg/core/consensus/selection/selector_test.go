package selection_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// TestSelectorRun tests that we can Run the selection
func TestSelectorRun(t *testing.T) {
	round := uint64(1)
	step := uint8(1)
	hash, _ := crypto.RandEntropy(32)
	msg := consensus.MockScoreMsg(t,
		&header.Header{
			BlockHash: hash,
			Round:     round,
			Step:      step,
		},
	)

	consensusTimeOut := 300 * time.Millisecond
	mockProxy := transactions.MockProxy{
		P: transactions.PermissiveProvisioner{},
	}
	emitter := consensus.MockEmitter(consensusTimeOut, mockProxy)

	cb := func(ctx context.Context) (bool, error) {
		packet := ctx.Value("Packet")
		require.NotNil(t, packet)

		if messageScore, ok := packet.(message.Score); ok {
			require.NotEmpty(t, messageScore.Score)

			require.Equal(t, msg.Payload(), messageScore)

			return true, nil
		}
		return false, errors.New("cb: failed to validate Score")
	}

	mockPhase := consensus.MockPhase(cb)
	sel := selection.New(mockPhase, emitter, consensusTimeOut)

	ctx, canc := context.WithTimeout(context.Background(), 2*time.Second)
	defer canc()

	queue := consensus.NewQueue()
	evChan := make(chan message.Message, 10)
	go func() {
		evChan <- msg
	}()

	phaseFn, err := sel.Run(ctx, queue, evChan, consensus.RoundUpdate{Round: round}, step)
	require.Nil(t, err)
	require.NotNil(t, phaseFn)

	_, err = phaseFn(ctx, queue, evChan, consensus.RoundUpdate{Round: round}, step+1)
	require.Nil(t, err)
}
