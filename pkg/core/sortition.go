package core

import (
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
)

func (b *Blockchain) sortition(role *role, threshold float64) (*bls.Sig, int, error) {
	// Construct message
	msg := append(b.currSeed, []byte(role.part)...)
	binary.LittleEndian.PutUint64(msg, role.round)
	binary.LittleEndian.PutUint64(msg, uint64(role.step))

	// Generate score and votes
	score, err := bls.Sign(b.BLSSecretKey, msg)
	if err != nil {
		return nil, 0, err
	}

	j := calcNormDist(threshold, b.stakeWeight, b.totalStakeWeight, score)

	return score, j, nil
}

func (b *Blockchain) verifySortition(score *bls.Sig, pk *bls.PublicKey, role *role, threshold float64, weight uint64) (int, error) {
	valid, err := b.verifyScore(score, pk, role)
	if err != nil {
		return 0, err
	}

	if valid {
		j := calcNormDist(threshold, weight, b.totalStakeWeight, score)
		return j, nil
	}

	return 0, nil
}

func (b *Blockchain) verifyScore(score *bls.Sig, pk *bls.PublicKey, role *role) (bool, error) {
	// Construct message
	msg := append(b.currSeed, []byte(role.part)...)
	binary.LittleEndian.PutUint64(msg, role.round)
	binary.LittleEndian.PutUint64(msg, uint64(role.step))

	if err := bls.Verify(pk, msg, score); err != nil {
		if err.Error() == "bls: Invalid Signature" {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
