package selection_test

import (
	"context"
	"errors"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// TestSelectorRun tests that we can Run the selection
func TestSelectorRun(t *testing.T) {
	queue := consensus.NewQueue()
	evChan := make(chan message.Message, 10)
	step := uint8(1)

	hash, err := crypto.RandEntropy(32)
	require.Nil(t, err)
	hdr := header.Mock()
	hdr.BlockHash = hash
	hdr.Round = 1

	// mock candidate
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	c := message.MakeCandidate(genesis, cert)

	se := message.MockScore(hdr, c)

	msg := message.New(topics.Score, se)

	consensusTimeOut := 100 * time.Millisecond

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

	ctx := context.Background()

	go func() {
		evChan <- msg
	}()

	phaseFn, err := sel.Run(ctx, queue, evChan, consensus.RoundUpdate{Round: uint64(1)}, step)
	require.Nil(t, err)
	require.NotNil(t, phaseFn)

	_, err = phaseFn(ctx, queue, evChan, consensus.RoundUpdate{Round: uint64(1)}, step+1)

	require.Nil(t, err)
}
