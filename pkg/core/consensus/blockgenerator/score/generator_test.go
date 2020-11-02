package score

import (
	"context"
	"encoding/binary"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/stretchr/testify/require"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// TestGenerate tests that we can run the score
func TestGenerate(t *testing.T) {
	round := uint64(1)
	step := uint8(1)

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../../../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	// creating the Helper

	d, _ := crypto.RandEntropy(32)
	k, _ := crypto.RandEntropy(32)
	idx, _ := crypto.RandEntropy(8)
	indexStoredBid := binary.LittleEndian.Uint64(idx)

	e, _ := consensus.StupidEmitter()
	scoreInstance := generator{
		Emitter:        e,
		scoreGenerator: e.Proxy.BlockGenerator(),
		d:              &common.JubJubCompressed{Data: d},
		k:              &common.BlsScalar{Data: k},
		indexStoredBid: indexStoredBid,
		threshold:      consensus.NewThreshold(),
	}

	// wiring the Gossip streamer to capture the gossiped messages

	ctx, canc := context.WithTimeout(context.Background(), 2*time.Second)
	defer canc()
	scoreProposal := scoreInstance.Generate(ctx, consensus.RoundUpdate{Round: round}, step)
	require.False(t, scoreProposal.IsEmpty())
}
