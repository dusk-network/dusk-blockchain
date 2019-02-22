package generation

import (
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// Refactor of code made by jules

// Block will generate a blockMsg and ScoreMsg if node is eligible.
func Block(ctx *user.Context) error {
	err := generateParams(ctx)
	if err != nil {
		return err
	}

	k, err := crypto.RandEntropy(32)
	if err != nil {
		return err
	}

	ctx.K = k

	// check threshold (eligibility)
	if ctx.Q <= ctx.Tau {
		return nil
	}

	// Generate ZkProof and Serialise
	// XXX: Prove may return error, so chaining like this may not be possible once zk implemented
	zkBytes, err := zkproof.Prove(ctx.X, ctx.Y, ctx.Z, ctx.M, ctx.K, ctx.Q, ctx.D).Bytes()
	if err != nil {
		return err
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
		ctx.Q,                      // score
		zkBytes,                    // zkproof
		candidateBlock.Header.Hash, // candidateHash
		ctx.Seed,                   // seed for this round // XXX(TOG): could we use round number/Block height?
	)
	if err != nil {
		return err
	}

	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		return err
	}

	msgScore, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	ctx.CandidateScoreChan <- msgScore

	// Create candidate msg
	pl2 := consensusmsg.NewCandidate(candidateBlock)
	sigEd2, err := ctx.CreateSignature(pl2)
	if err != nil {
		return err
	}

	msgCandidate, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd2, []byte(*ctx.Keys.EdPubKey), pl2)
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

// generate M, X, Y, Z, Q
func generateParams(ctx *user.Context) error {
	// XXX: generating X, Y, Z in this way is in-efficient. Passing the parameters in directly from previous computation is better.
	// Wait until, specs for this has been semi-finalised
	M, err := GenerateM(ctx.K)
	if err != nil {
		return err
	}
	X, err := GenerateX(ctx.D, ctx.K)
	if err != nil {
		return err
	}
	Y, err := GenerateY(ctx.D, ctx.LastHeader.Seed, ctx.K)
	if err != nil {
		return err
	}
	Z, err := generateZ(ctx.LastHeader.Seed, ctx.K)
	if err != nil {
		return err
	}
	Q, err := Score(ctx.D, Y)
	if err != nil {
		return err
	}

	ctx.M = M
	ctx.X = X
	ctx.Y = Y
	ctx.Z = Z
	ctx.Q = Q

	return nil

}

// GenerateM will return M so that M = H(k)
func GenerateM(k []byte) ([]byte, error) {
	M, err := hash.Sha3256(k)
	if err != nil {
		return nil, err
	}

	return M, nil
}

// GenerateX will return X so that X = H(d, M)
func GenerateX(d uint64, k []byte) ([]byte, error) {

	M, err := GenerateM(k)

	dM := make([]byte, 8, 40)

	binary.LittleEndian.PutUint64(dM, d)

	dM = append(dM, M...)

	X, err := hash.Sha3256(dM)
	if err != nil {
		return nil, err
	}

	return X, nil
}

// GenerateY will return Y so that Y = H(S, X)
func GenerateY(d uint64, S, k []byte) ([]byte, error) {

	X, err := GenerateX(d, k)
	if err != nil {
		return nil, err
	}

	SX := make([]byte, 0, 64) // X = 32 , prevSeed = 32
	SX = append(SX, S...)
	SX = append(SX, X...)

	Y, err := hash.Sha3256(SX)
	if err != nil {
		return nil, err
	}

	return Y, nil
}

// Z = H(S, M)
func generateZ(S, k []byte) ([]byte, error) {

	M, err := GenerateM(k)

	SM := make([]byte, 0, 64) // M = 32 , prevSeed = 32
	SM = append(SM, S...)
	SM = append(SM, M...)

	Z, err := hash.Sha3256(SM)
	if err != nil {
		return nil, err
	}

	return Z, nil
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
