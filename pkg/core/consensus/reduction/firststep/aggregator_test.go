package firststep

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestSuccessfulAggro tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestSuccessfulAggro(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(eb, rbus, 10, 1*time.Second)
	evs := hlp.Spawn(hash)

	res := make(chan error, 1)
	test := func(hash []byte, svs ...*message.StepVotes) {
		assert.Equal(t, hlp.Step(), svs[0].Step)
		res <- hlp.Verify(hash, *svs[0], hlp.Step())
	}

	aggregator := newAggregator(test, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		if !assert.NoError(t, aggregator.collectVote(ev)) {
			t.FailNow()
		}
	}

	err := <-res
	assert.NoError(t, err)
}

// TestInvalidBlock tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestInvalidBlock(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(eb, rbus, 10, 1*time.Second)
	hlp.FailOnVerification(true)
	evs := hlp.Spawn(hash)

	res := make(chan struct{}, 1)
	test := func(hash []byte, svs ...*message.StepVotes) {
		assert.Equal(t, emptyHash[:], hash)
		assert.Equal(t, 0, len(svs))
		res <- struct{}{}
	}

	aggregator := newAggregator(test, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		if !assert.NoError(t, aggregator.collectVote(ev)) {
			t.FailNow()
		}
	}

	<-res
}

// Test that a valid stepvotes is produced when a candidate block for
// a given hash is not found.
func TestCandidateNotFound(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(eb, rbus, 10, 1*time.Second)
	hlp.FailOnFetching(true)
	evs := hlp.Spawn(hash)

	res := make(chan struct{}, 1)
	test := func(hash []byte, svs ...*message.StepVotes) {
		assert.Equal(t, emptyHash[:], hash)
		assert.Equal(t, 0, len(svs))
		res <- struct{}{}
	}

	aggregator := newAggregator(test, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		if !assert.NoError(t, aggregator.collectVote(ev)) {
			t.FailNow()
		}
	}

	<-res
}
