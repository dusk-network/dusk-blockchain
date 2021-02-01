// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus

import (
	"math/big"
)

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

// Exceeds checks whether the Threshold exceeds a given score.
func (t *Threshold) Exceeds(score []byte) bool {
	scoreInt := big.NewInt(0).SetBytes(score)
	return scoreInt.Cmp(t.limit) == -1
}
