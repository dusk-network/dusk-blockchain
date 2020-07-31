package firststep

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestSuccessfulAggro tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestSuccessfulAggro(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := KickstartConcurrent(eb, rbus, 10, 1*time.Second)
	evs := hlp.Spawn(hash)

	res := make(chan reduction.HaltMsg, 1)

	aggregator := newAggregator(res, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		if !assert.NoError(t, aggregator.collectVote(ev)) {
			t.FailNow()
		}
	}

	m := <-res
	assert.Equal(t, hlp.Step(), m.Sv[0].Step)
	assert.NoError(t, hlp.Verify(m.Hash, *m.Sv[0], hlp.Step()))
}

// TestInvalidBlock tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestInvalidBlock(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := KickstartConcurrent(eb, rbus, 10, 1*time.Second)
	hlp.FailOnVerification(true)
	evs := hlp.Spawn(hash)

	res := make(chan reduction.HaltMsg, 1)

	aggregator := newAggregator(res, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		if !assert.NoError(t, aggregator.collectVote(ev)) {
			t.FailNow()
		}
	}

	m := <-res
	assert.Equal(t, emptyHash[:], m.Hash)
	assert.Equal(t, 0, len(m.Sv))
}

// Test that a valid stepvotes is produced when a candidate block for
// a given hash is not found.
func TestCandidateNotFound(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := KickstartConcurrent(eb, rbus, 10, 1*time.Second)
	hlp.FailOnFetching(true)
	evs := hlp.Spawn(hash)

	res := make(chan reduction.HaltMsg, 1)

	aggregator := newAggregator(res, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		if !assert.NoError(t, aggregator.collectVote(ev)) {
			t.FailNow()
		}
	}

	m := <-res
	assert.Equal(t, emptyHash[:], m.Hash)
	assert.Equal(t, 0, len(m.Sv))
}
