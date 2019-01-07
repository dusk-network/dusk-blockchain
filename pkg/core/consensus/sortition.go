package consensus

import (
	"encoding/binary"
	"math/big"

	"gonum.org/v1/gonum/stat/distuv"
)

func sortition(ctx *Context, role *role) error {
	// Construct message
	msg := append(ctx.Seed, []byte(role.part)...)
	binary.LittleEndian.PutUint64(msg, role.round)
	binary.LittleEndian.PutUint64(msg, uint64(role.step))

	// Generate score and votes
	score, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, msg)
	if err != nil {
		return err
	}

	votes := calcVotes(ctx.Threshold, ctx.weight, ctx.W, score)
	ctx.votes = votes
	ctx.Score = score

	return nil
}

func verifySortition(ctx *Context, score, pk []byte, role *role, stake uint64) (int, error) {
	valid, err := verifyScore(ctx, score, pk, role)
	if err != nil {
		return 0, err
	}

	if valid {
		votes := calcVotes(ctx.Threshold, stake, ctx.W, score)
		return votes, nil
	}

	return 0, nil
}

func verifyScore(ctx *Context, score, pk []byte, role *role) (bool, error) {
	// Construct message
	msg := append(ctx.Seed, []byte(role.part)...)
	binary.LittleEndian.PutUint64(msg, role.round)
	binary.LittleEndian.PutUint64(msg, uint64(role.step))

	if err := ctx.BLSVerify(pk, msg, score); err != nil {
		if err.Error() == "bls: Invalid Signature" {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func calcVotes(threshold float64, stake, totalStake uint64, score []byte) int {
	// Calculate probability (sigma)
	p := threshold / float64(totalStake)

	// Calculate score divided by 2^256
	var lenHash, _ = new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
	scoreNum := new(big.Int).SetBytes(score)
	target, _ := new(big.Rat).SetFrac(scoreNum, lenHash).Float64()

	votes := float64(0.0)
	for {
		dist := distuv.Normal{
			Mu:    float64(stake),
			Sigma: p,
		}

		p1 := dist.Prob(votes)
		p2 := dist.Prob(votes + 1.0)
		if p1 > target && target > p2 {
			break
		}

		votes += 1.0
	}

	return int(votes)
}
