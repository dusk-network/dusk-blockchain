package firststep

import (
	"testing"
	"time"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
)

var round = uint64(1)
var step = uint8(2)

var ttest = map[string]struct {
	setup func(*Helper)
	tCb   func(*require.Assertions, *Helper, *result)
}{
	"Test Successful Aggregation": {
		setup: func(hlp *Helper) {},
		tCb: func(require *require.Assertions, hlp *Helper, res *result) {
			require.NotNil(res)
			require.NoError(hlp.Verify(res.Hash, res.SV, round, step))
		},
	},

	"Test Invalid Block": {
		setup: func(hlp *Helper) {
			hlp.FailOnVerification(true)
		},
		tCb: func(require *require.Assertions, hlp *Helper, res *result) {
			require.Equal(emptyHash[:], res.Hash)
			require.True(res.SV.IsEmpty())
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
}

// TestAggregation tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestAggregation(t *testing.T) {
	t.Parallel()
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
			aggregator := newAggregator(hlp.Handler, hlp.RPCBus)

			// running test-specific setup on the Helper
			tt.setup(hlp)

			// creating the messages
			evs := hlp.Spawn(hash, round, step)

			// sending Reduction messages to the aggregator
			var res *result
			for _, ev := range evs {
				var err error
				res, err = aggregator.collectVote(ev)
				require.Nil(err)
				if res != nil {
					break
				}
			}

			tt.tCb(require, hlp, res)
		})
	}
}
