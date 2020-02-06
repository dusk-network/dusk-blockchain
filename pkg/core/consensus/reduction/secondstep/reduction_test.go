package secondstep

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

func TestSecondStep(t *testing.T) {
	hlp, hash := Kickstart(50, 1*time.Second)

	// Generate first StepVotes
	svs := message.GenVotes(hash, 1, 2, hlp.Keys, hlp.P)

	// Start the first step
	if err := hlp.ActivateReduction(hash, *svs[0]); err != nil {
		t.Fatal(err)
	}

	// Send events
	hlp.SendBatch(hash)

	// Wait for resulting Agreement
	agMsg := <-hlp.AgreementChan
	ag := agMsg.Payload().(message.Agreement)

	// Ensure we get a regeneration message
	<-hlp.RestartChan

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, *ag.VotesPerStep[0], 0))
	assert.NoError(t, hlp.Verify(hash, *ag.VotesPerStep[1], 1))

	// Timeout should be the same
	assert.Equal(t, 1*time.Second, hlp.Reducer.(*Reducer).timeOut)
}

func TestSecondStepAfterFailure(t *testing.T) {
	timeOut := 1000 * time.Millisecond
	hlp, hash := Kickstart(50, timeOut)

	// Start the first step
	if err := hlp.ActivateReduction(hash, message.StepVotes{}); err != nil {
		t.Fatal(err)
	}

	// Send events
	hlp.SendBatch(hash)

	// Ensure we get a regeneration message
	<-hlp.RestartChan

	// Make sure no agreement message is sent
	select {
	case <-hlp.AgreementChan:
		t.Fatal("not supposed to construct an agreement if the first StepVotes is nil")
	case <-time.After(time.Second * 1):
		// Ensure timeout was doubled
		assert.Equal(t, timeOut*2, hlp.Reducer.(*Reducer).timeOut)
		// Success
	}
}
