// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/stretchr/testify/assert"
)

var score = &common.BlsScalar{Data: []byte{
	120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
	120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
	120, 120, 120, 120, 120, 120,
}}

func TestThresholdCheck(t *testing.T) {
	threshold := consensus.NewThreshold()
	// The standard Threshold is set to [170, 170, ...], so the threshold should exceed the score.
	assert.True(t, threshold.Exceeds(score))
}

func TestLowerThreshold(t *testing.T) {
	threshold := consensus.NewThreshold()
	// lower threshold now (it should be divided by 2, bringing it to [85, 85, ...])
	threshold.Lower()
	// threshold no longer exceeds the score
	assert.False(t, threshold.Exceeds(score))
}

func TestResetThreshold(t *testing.T) {
	threshold := consensus.NewThreshold()

	threshold.Lower()
	threshold.Reset()

	assert.True(t, threshold.Exceeds(score))
}
