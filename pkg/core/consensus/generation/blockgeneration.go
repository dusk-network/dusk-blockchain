package generation

import (
	"math/big"
	"sync/atomic"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// Refactor of code made by jules

// Block will generate a blockMsg and ScoreMsg if node is eligible.
func Block(ctx *user.Context) error {
	// Generate ZkProof and Serialise
	// XXX: Prove may return error, so chaining like this may not be possible once zk implemented
	dScalar := zkproof.Uint64ToScalar(ctx.D)
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(ctx.Seed)
	numBids := len(ctx.PubList)
	if numBids > 10 {
		numBids = 10
	}

	bids := ctx.PubList.GetRandomBids(numBids)
	pubList := make([]ristretto.Scalar, len(bids))
	for i, bid := range bids {
		bidScalar := zkproof.BytesToScalar(bid[:])
		pubList[i] = bidScalar
	}

	proof, q, z, pL := zkproof.Prove(dScalar, ctx.K, seedScalar, pubList)

	score := big.NewInt(0).SetBytes(q).Uint64()
	if score < ctx.Tau {
		return nil
	}

	// Seed is the candidate signature of the previous seed
	seed, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.LastHeader.Seed)
	if err != nil {
		return err
	}

	ctx.Seed = seed

	// Generate candidate block
	candidateBlock, err := newCandidateBlock(ctx)
	if err != nil {
		return err
	}

	// Create score msg
	pl, err := consensusmsg.NewCandidateScore(
		q,                          // score
		proof,                      // zkproof
		z,                          // Identity hash
		candidateBlock.Header.Hash, // candidateHash
		ctx.Seed,                   // seed for this round // XXX(TOG): could we use round number/Block height?
		pL,
	)
	if err != nil {
		return err
	}

	sigEd, err := ctx.CreateSignature(pl, atomic.LoadUint32(&ctx.BlockStep))
	if err != nil {
		return err
	}

	msgScore, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		atomic.LoadUint32(&ctx.BlockStep), sigEd, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return err
	}

	ctx.CandidateScoreChan <- msgScore

	// Create candidate msg
	pl2 := consensusmsg.NewCandidate(candidateBlock)
	sigEd2, err := ctx.CreateSignature(pl2, atomic.LoadUint32(&ctx.BlockStep))
	if err != nil {
		return err
	}

	msgCandidate, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		atomic.LoadUint32(&ctx.BlockStep), sigEd2, ctx.Keys.EdPubKeyBytes(), pl2)
	if err != nil {
		return err
	}

	if err := ctx.SendMessage(ctx.Magic, msgCandidate); err != nil {
		return err
	}

	// Set value on our context
	ctx.BlockHash = candidateBlock.Header.Hash
	ctx.CandidateBlock = candidateBlock

	return nil
}

func newCandidateBlock(ctx *user.Context) (*block.Block, error) {

	candidateBlock := block.NewBlock()

	candidateBlock.SetPrevBlock(ctx.LastHeader)

	candidateBlock.Header.Seed = ctx.Seed
	candidateBlock.Header.Height = ctx.Round
	candidateBlock.Header.Timestamp = time.Now().Unix()

	// XXX: Generate coinbase/reward beforehand
	// Coinbase is still not decided
	txs := ctx.GetAllTXs()
	for _, tx := range txs {
		candidateBlock.AddTx(tx)
	}
	err := candidateBlock.SetRoot()
	if err != nil {
		return nil, err
	}

	err = candidateBlock.SetHash()
	if err != nil {
		return nil, err
	}

	return candidateBlock, nil
}
