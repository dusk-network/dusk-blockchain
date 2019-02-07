package sortition

import (
	"encoding/binary"
	"errors"
	"math/big"
	"strings"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gonum.org/v1/gonum/stat/distuv"
)

// Role contains information about a node performing consensus.
type Role struct {
	Part  string
	Round uint64
	Step  uint8
}

var (
	i       = big.NewInt(2)
	e       = big.NewInt(256)
	lenHash = i.Exp(i, e, nil)

	// MinimumStake is the minimum stake needed to be eligible for sortition
	MinimumStake uint64 = 100
)

// Prove will run the sortition function for the passed context and role.
func Prove(seed []byte, sk *bls.SecretKey, pk *bls.PublicKey, role *Role,
	threshold, weight, totalWeight uint64) (uint64, []byte, *prerror.PrError) {
	// Construct message
	msg := append(seed, []byte(role.Part)...)
	round := make([]byte, 8)
	binary.LittleEndian.PutUint64(round, role.Round)
	msg = append(msg, round...)
	msg = append(msg, byte(role.Step))

	// Generate score
	sig, err := bls.Sign(sk, pk, msg)
	if err != nil {
		return 0, nil, prerror.New(prerror.High, err)
	}

	// Compress score
	score := sig.Compress()

	// Generate votes
	votes, err := calcVotes(threshold, weight, totalWeight, score)
	if err != nil {
		return 0, nil, prerror.New(prerror.Low, err)
	}

	return votes, score, nil
}

// Verify will run the sortition function for another node's information.
func Verify(seed, score, pk []byte, role *Role, threshold, stake,
	totalWeight uint64) (uint64, *prerror.PrError) {
	if scoreErr := verifyScore(seed, score, pk, role); scoreErr != nil {
		return 0, scoreErr
	}

	votes, err := calcVotes(threshold, stake, totalWeight, score)
	if err != nil {
		return 0, prerror.New(prerror.Low, err)
	}

	return votes, nil
}

func verifyScore(seed, score, pkBytes []byte, role *Role) *prerror.PrError {
	// Construct message
	msg := append(seed, []byte(role.Part)...)
	round := make([]byte, 8)
	binary.LittleEndian.PutUint64(round, role.Round)
	msg = append(msg, round...)
	msg = append(msg, byte(role.Step))

	// And verify
	pk := &bls.PublicKey{}
	pk.Unmarshal(pkBytes)
	apk := bls.NewApk(pk)

	sig := &bls.Signature{}
	sig.Decompress(score)

	if err := bls.Verify(apk, msg, sig); err != nil {
		if strings.Contains(err.Error(), "Invalid Signature") {
			return prerror.New(prerror.Low, err)
		}

		return prerror.New(prerror.High, err)
	}

	return nil
}

func calcVotes(threshold, stake, totalStake uint64, score []byte) (uint64, error) {
	// Sanity checks
	if threshold > totalStake {
		return 0, errors.New("threshold size should not exceed maximum stake weight")
	}

	if stake < MinimumStake {
		return 0, errors.New("stake not big enough")
	}

	// Calculate probability (sigma)
	p := float64(threshold) / float64(totalStake)
	if p > 1 || p < 0 {
		return 0, errors.New("p should be between 0 and 1")
	}

	// Calculate score divided by 2^256
	scoreNum := new(big.Int).SetBytes(score[:32])
	target, _ := new(big.Rat).SetFrac(scoreNum, lenHash).Float64()

	// Set up the distribution
	dist := distuv.Normal{
		Mu:    float64(stake / 100),
		Sigma: p,
	}

	// Calculate votes
	pos := float64(0.0)
	votes := uint64(0)
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
