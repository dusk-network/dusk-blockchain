package consensus

import (
	"encoding/binary"
	"errors"
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

	votes, err := calcVotes(ctx.Threshold, ctx.weight, ctx.W, score)
	if err != nil {
		return err
	}

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
		votes, err := calcVotes(ctx.Threshold, stake, ctx.W, score)
		if err != nil {
			return 0, err
		}

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

func calcVotes(threshold, stake, totalStake uint64, score []byte) (int, error) {
	if threshold > totalStake {
		return 0, errors.New("threshold size should not exceed maximum stake weight")
	}

	if stake < 100 {
		return 0, errors.New("stake not big enough")
	}

	// Calculate probability (sigma)
	p := float64(threshold) / float64(totalStake)
	if p > 1 || p < 0 {
		return 0, errors.New("p should be between 0 and 1")
	}

	// Calculate score divided by 2^256
	var lenHash, _ = new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
	scoreNum := new(big.Int).SetBytes(score)
	target, _ := new(big.Rat).SetFrac(scoreNum, lenHash).Float64()

	dist := distuv.Normal{
		Mu:    float64(stake / 100),
		Sigma: p,
	}

	pos := float64(0.0)
	votes := 0
	for {
		p1 := dist.Prob(pos)
		p2 := dist.Prob(pos + 0.01)
		if target >= p1 && target <= p2 {
			break
		}

		pos += 0.01
		votes++
	}

	return votes, nil
}
