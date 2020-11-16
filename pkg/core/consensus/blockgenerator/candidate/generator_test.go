package candidate_test

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	hlp := candidate.NewHelper(50, time.Second)

	_, pubKey := transactions.MockKeys()
	gen := candidate.New(hlp.Emitter, pubKey)

	ctx := context.Background()

	hash, _ := crypto.RandEntropy(32)
	hdr := header.Header{
		Round:     uint64(1),
		Step:      uint8(1),
		BlockHash: hash,
		PubKeyBLS: hlp.ThisSender,
	}
	scr := message.MockScoreProposal(hdr)
	ru := consensus.MockRoundUpdate(uint64(1), hlp.P)
	_, err := gen.GenerateCandidateMessage(ctx, scr, ru, uint8(1))
	require.NoError(t, err)
}
