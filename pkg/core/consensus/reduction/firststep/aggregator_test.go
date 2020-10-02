package firststep

import (
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestSuccessfulAggro tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestSuccessfulAggro(t *testing.T) {
	step := uint8(2)
	round := uint64(1)
	messageToSpawn := 3
	eb, rbus := eventbus.New(), rpcbus.New()

	hlp := NewHelper(eb, rbus, messageToSpawn+1, 1*time.Second, true)

	hash, _ := crypto.RandEntropy(32)
	evs := hlp.Spawn(hash, round, step)

	aggregator := newAggregator(hlp.Handler, rbus)

	var res *result
	for _, ev := range evs {
		var err error
		res, err = aggregator.collectVote(ev)
		require.Nil(t, err)
		if res != nil {
			break
		}

	}

	assert.NotNil(t, res)

	assert.NoError(t, hlp.Verify(res.Hash, res.SV, round, step))
}

// TestInvalidBlock tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestInvalidBlock(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	messageToSpawn := 3
	step := uint8(2)
	round := uint64(1)

	eb, rbus := eventbus.New(), rpcbus.New()
	hash, _ := crypto.RandEntropy(32)

	hlp := NewHelper(eb, rbus, messageToSpawn+1, 1*time.Second, true)

	hlp.FailOnVerification(true)
	evs := hlp.Spawn(hash, round, step)

	aggregator := newAggregator(hlp.Handler, rbus)

	var res *result
	for _, ev := range evs {
		var err error
		res, err = aggregator.collectVote(ev)
		require.Nil(t, err)
		if res != nil {
			break
		}

	}

	assert.Equal(t, emptyHash[:], res.Hash)
	assert.True(t, res.SV.IsEmpty())
}

// Test that a valid stepvotes is produced when a candidate block for
// a given hash is not found.
func TestCandidateNotFound(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	messageToSpawn := 3
	step := uint8(2)
	round := uint64(1)

	eb, rbus := eventbus.New(), rpcbus.New()
	hash, _ := crypto.RandEntropy(32)

	hlp := NewHelper(eb, rbus, messageToSpawn+1, 1*time.Second, true)
	hlp.FailOnFetching(true)
	evs := hlp.Spawn(hash, round, step)

	aggregator := newAggregator(hlp.Handler, hlp.RBus)

	var res *result
	for _, ev := range evs {
		var err error
		res, err = aggregator.collectVote(ev)
		require.Nil(t, err)
		if res != nil {
			break
		}

	}

	assert.Equal(t, emptyHash[:], res.Hash)
	assert.True(t, res.SV.IsEmpty())
}
