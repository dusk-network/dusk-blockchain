package score

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/candidate"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/stretchr/testify/require"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// TestSelectorRun tests that we can Run the selection
func TestScoreStepRun(t *testing.T) {
	round := uint64(1)
	step := uint8(1)

	consensusTimeOut := 300 * time.Millisecond
	mockProxy := transactions.MockProxy{
		P: transactions.PermissiveProvisioner{},
	}
	emitter := consensus.MockEmitter(consensusTimeOut, mockProxy)

	cb := func(ctx context.Context) (bool, error) {
		return true, nil
	}

	d, _ := crypto.RandEntropy(32)
	k, _ := crypto.RandEntropy(32)
	edPk, _ := crypto.RandEntropy(32)

	mockPhase := consensus.MockPhase(cb)
	blockGen := candidate.New(emitter, &transactions.PublicKey{})

	scoreInstance := Phase{
		Emitter:   emitter,
		bg:        emitter.Proxy.BlockGenerator(),
		d:         d,
		k:         k,
		edPk:      edPk,
		threshold: consensus.NewThreshold(),
		next:      mockPhase,
		generator: blockGen,
	}

	ctx, canc := context.WithTimeout(context.Background(), 2*time.Second)
	defer canc()

	phaseFn, err := scoreInstance.Run(ctx, nil, nil, consensus.RoundUpdate{Round: round}, step)
	require.Nil(t, err)
	require.NotNil(t, phaseFn)

	_, err = phaseFn(ctx, nil, nil, consensus.RoundUpdate{Round: round}, step+1)
	require.Nil(t, err)
}
