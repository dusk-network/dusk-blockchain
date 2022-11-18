// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package reduction

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
)

var (
	round = uint64(1)
	step  = uint8(2)
)

var ttest = map[string]struct {
	setup func(*Helper)
	tCb   func(*require.Assertions, *Helper, *Result)
}{
	"Test Successful Aggregation": {
		setup: func(hlp *Helper) {},
		tCb: func(require *require.Assertions, hlp *Helper, res *Result) {
			require.NotNil(res)
			require.NoError(hlp.Verify(res.Hash, res.SV, round, step))
		},
	},

	/*
		// TODO: move those into the firststep
		"Test Invalid Block": {
			setup: func(hlp *Helper) {
				hlp.FailOnVerification(true)
			},
			tCb: func(require *require.Assertions, hlp *Helper, res *Result) {
				require.True(res.IsEmpty())
			},
		},

		"Test Candidate Not Found": {
			setup: func(hlp *Helper) {
				hlp.FailOnFetching(true)
			},
			tCb: func(require *require.Assertions, hlp *Helper, res *result) {
				require.Equal(emptyHash[:], res.Hash)
				require.True(res.SV.IsEmpty())
			},
		},
	*/
}

// TestAggregation tests that upon collection of a quorum of events, a valid StepVotes get produced.
func TestAggregation(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	messageToSpawn := 3

	for testName, tt := range ttest {
		t.Run(testName, func(t *testing.T) {
			// making sure that parallelism does not interfere with the test
			tt := tt
			// creting the require instance from this subtest
			require := require.New(t)
			// setting up the helper and the aggregator
			hlp := NewHelper(messageToSpawn+1, 1*time.Second)
			aggregator := NewAggregator(hlp.Handler)

			// running test-specific setup on the Helper
			tt.setup(hlp)

			// creating the messages
			evs := hlp.Spawn(hash, round, step)

			// sending Reduction messages to the aggregator
			var res *Result

			for _, ev := range evs {
				// if the aggregator returns a result, the quorum has been
				// reached. Otherwise it returns nil
				if res = aggregator.CollectVote(ev); res != nil {
					break
				}
			}

			tt.tCb(require, hlp, res)
		})
	}
}

func TestAggregation2(t *testing.T) {
	hash, _ := hex.DecodeString("b70189c7e7a347989f4fbc1205ce612f755dfc489ecf28f9f883800acf078bd5")

	round := uint64(1)
	step := uint8(1)

	for testName, tt := range ttest {
		t.Run(testName, func(t *testing.T) {
			// making sure that parallelism does not interfere with the test
			tt := tt
			// creting the require instance from this subtest

			// setting up the helper and the aggregator
			hlp := NewHelper(5, 1*time.Second)
			aggregator := NewAggregator(hlp.Handler)

			// running test-specific setup on the Helper
			tt.setup(hlp)

			// creating the messages
			evs := make([]message.Reduction, 0)
			for i := 0; i < 4; i++ {
				ev := message.MockReduction(hash, round, step, hlp.ProvisionersKeys, i)
				evs = append(evs, ev)
			}

			// sending Reduction messages to the aggregator
			var res *Result

			for _, ev := range evs {
				// if the aggregator returns a result, the quorum has been
				// reached. Otherwise it returns nil
				if res = aggregator.CollectVote(ev); res != nil {

					t.Logf("%s %d", hex.EncodeToString(res.Hash), res.SV.BitSet)
					break
				}
			}
		})
	}
}
