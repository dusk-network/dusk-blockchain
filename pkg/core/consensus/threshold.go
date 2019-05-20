package consensus

import "math/big"

type Threshold struct {
	limit *big.Int
}

func NewThreshold() *Threshold {
	limit, _ := big.NewInt(0).SetString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 16)
	return &Threshold{
		limit: limit,
	}
}

func (t *Threshold) Reset() {
	t.limit, _ = big.NewInt(0).SetString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 16)
}

func (t *Threshold) Lower() {
	t.limit.Div(t.limit, big.NewInt(2))
}

func (t *Threshold) Exceeds(score []byte) bool {
	scoreInt := big.NewInt(0).SetBytes(score)
	if scoreInt.Cmp(t.limit) == -1 {
		return false
	}

	return true
}
