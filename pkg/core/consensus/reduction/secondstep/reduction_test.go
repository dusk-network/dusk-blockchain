package secondstep

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/stretchr/testify/assert"
)

func TestSecondStep(t *testing.T) {
	hlp, hash := Kickstart(50)

	// Generate first StepVotes
	svs := agreement.GenVotes(hash, 1, 1, hlp.Keys, hlp.P.CreateVotingCommittee(1, 1, 50))

	// Start the first step
	if err := hlp.ActivateReduction(hash, svs[0]); err != nil {
		t.Fatal(err)
	}

	// Send events (most probably this requires a Forward)
	hlp.SendBatch(hash)

	// Wait for resulting Agreement
	agBuf := <-hlp.AgreementChan

	// Ensure we get a regeneration message
	<-hlp.RegenChan

	// Retrieve Agreement
	ag := agreement.New(header.Header{})
	assert.NoError(t, agreement.Unmarshal(&agBuf, ag))

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, ag.VotesPerStep[0], 1))
	assert.NoError(t, hlp.Verify(hash, ag.VotesPerStep[1], 2))
}

func TestSecondStepAfterFailure(t *testing.T) {
	hlp, hash := Kickstart(50)

	// Start the first step
	if err := hlp.ActivateReduction(hash, nil); err != nil {
		t.Fatal(err)
	}

	// Send events
	hlp.SendBatch(hash)

	// Ensure we get a regeneration message
	<-hlp.RegenChan

	// Make sure no agreement message is sent
	select {
	case <-hlp.AgreementChan:
		t.Fatal("not supposed to construct an agreement if the first StepVotes is nil")
	case <-time.After(time.Second * 1):
		// Success
	}
}

func TestSecondStepTimeOut(t *testing.T) {
	hlp, hash := Kickstart(50)

	// Start the first step
	if err := hlp.ActivateReduction(hash, nil); err != nil {
		t.Fatal(err)
	}

	// Ensure we get a regeneration message
	<-hlp.RegenChan

	// Make sure no agreement message is sent
	select {
	case <-hlp.AgreementChan:
		t.Fatal("not supposed to construct an agreement if the first StepVotes is nil")
	case <-time.After(time.Second * 1):
		// Success
	}
}
