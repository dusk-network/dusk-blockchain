package consensus

import (
	"encoding/binary"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// Refactor of code made by jules

// GenerateBlock will generate a blockMsg and ScoreMsg
// if node is eligible.
func GenerateBlock(ctx *Context, k []byte) (*payload.MsgScore, *payload.MsgBlock, error) {

	err := generateParams(ctx)
	if err != nil {
		return nil, nil, err
	}

	// check threshold (eligibility)
	if ctx.Q <= ctx.Tau {
		return nil, nil, errors.New("Score is less than tau (threshold)")
	}

	// Generate ZkProof and Serialise
	// XXX: Prove may return error, so chaining like this may not be possible once zk implemented
	zkBytes, err := zkproof.Prove(ctx.X, ctx.Y, ctx.Z, ctx.M, ctx.k, ctx.Q, ctx.d).Bytes()
	if err != nil {
		return nil, nil, err
	}

	// Generate candidate block
	candidateBlock, err := newCandidateBlock(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Sign msg score content with Ed25519
	buf := make([]byte, 0, 40)
	binary.LittleEndian.PutUint64(buf, ctx.Q)
	buf = append(buf, zkBytes...)
	buf = append(buf, candidateBlock.Header.Hash...)
	buf = append(buf, ctx.LastHeader.Seed...)

	sig := ctx.EDSign(ctx.Keys.EdSecretKey, buf)

	// Create score msg
	msgScore, err := payload.NewMsgScore(
		ctx.Q,                      // score
		zkBytes,                    // zkproof
		candidateBlock.Header.Hash, // candidateHash
		sig,                        // sig
		[]byte(*ctx.Keys.EdPubKey), // pubKey
		ctx.Seed,                   // seed for this round // XXX(TOG): could we use round number/Block height?
	)

	// Create block msg
	msgBlock := payload.NewMsgBlock(candidateBlock)

	return msgScore, msgBlock, nil
}

// generate M, X, Y, Z, Q
func generateParams(ctx *Context) error {
	// XXX: generating X, Y, Z in this way is in-efficient. Passing the parameters in directly from previous computation is better.
	// Wait until, specs for this has been semi-finalised
	M, err := generateM(ctx.k)
	if err != nil {
		return err
	}
	X, err := generateX(ctx.d, ctx.k)
	if err != nil {
		return err
	}
	Y, err := generateY(ctx.d, ctx.LastHeader.Seed, ctx.k)
	if err != nil {
		return err
	}
	Z, err := generateZ(ctx.LastHeader.Seed, ctx.k)
	if err != nil {
		return err
	}
	Q, err := GenerateScore(ctx.d, Y)
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

// M = H(k)
func generateM(k []byte) ([]byte, error) {
	M, err := hash.Sha3256(k)
	if err != nil {
		return nil, err
	}

	return M, nil
}

// X = H(d, M)
func generateX(d uint64, k []byte) ([]byte, error) {

	M, err := generateM(k)

	dM := make([]byte, 8, 40)

	binary.LittleEndian.PutUint64(dM, d)

	dM = append(dM, M...)

	X, err := hash.Sha3256(dM)
	if err != nil {
		return nil, err
	}

	return X, nil
}

// Y = H(S, X)
func generateY(d uint64, S, k []byte) ([]byte, error) {

	X, err := generateX(d, k)
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

	M, err := generateM(k)

	SM := make([]byte, 0, 64) // M = 32 , prevSeed = 32
	SM = append(SM, S...)
	SM = append(SM, M...)

	Z, err := hash.Sha3256(SM)
	if err != nil {
		return nil, err
	}

	return Z, nil
}

func newCandidateBlock(ctx *Context) (*payload.Block, error) {

	candidateBlock := payload.NewBlock()

	err := candidateBlock.SetPrevBlock(ctx.LastHeader)
	if err != nil {
		return nil, err
	}

	// Seed is the candidate signature of the previous seed
	seed, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.LastHeader.Seed)
	if err != nil {
		return nil, err
	}

	err = candidateBlock.SetSeed(seed)
	if err != nil {
		return nil, err
	}

	candidateBlock.SetTime(time.Now().Unix())

	// XXX: Generate coinbase/reward beforehand
	// Coinbase is still not decided
	txs := ctx.GetAllTXs()
	for _, tx := range txs {
		candidateBlock.AddTx(tx)
	}
	err = candidateBlock.SetRoot()
	if err != nil {
		return nil, err
	}

	err = candidateBlock.SetHash()
	if err != nil {
		return nil, err
	}

	return candidateBlock, nil
}
