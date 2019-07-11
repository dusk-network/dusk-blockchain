package consensus

import "math/big"

// Threshold is a number which proof scores should be compared against.
// If a proof score does not exceed the Threshold value, it should be discarded.
type Threshold struct {
	limit *big.Int
}

// NewThreshold returns an initialized Threshold.
func NewThreshold() *Threshold {
	limit, _ := big.NewInt(0).SetString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 16)
	return &Threshold{
		limit: limit,
	}
}

// Reset the Threshold to its normal lower limit.
func (t *Threshold) Reset() {
	t.limit, _ = big.NewInt(0).SetString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 16)
}

// Lower the Threshold by cutting it in half.
func (t *Threshold) Lower() {
	t.limit.Div(t.limit, big.NewInt(2))
}

// Exceeds checks whether a passed score is higher or lower than the Threshold.
func (t *Threshold) Exceeds(score []byte) bool {
	scoreInt := big.NewInt(0).SetBytes(score)
	if scoreInt.Cmp(t.limit) == -1 {
		return false
	}

	return true
}
