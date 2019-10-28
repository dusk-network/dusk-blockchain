package secondstep

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestSecondStep(t *testing.T) {
	hlp, hash := startReductionTest()

	// Generate first StepVotes
	svs := agreement.GenVotes(hash, 1, 1, hlp.Keys, hlp.P.CreateVotingCommittee(1, 1, 50))

	// Start the first step
	if err := hlp.StartReduction(svs[0]); err != nil {
		t.Fatal(err)
	}

	// Send events
	hlp.SendBatch(hash, 1, 2)

	// Wait for resulting Agreement
	var agBuf bytes.Buffer
	select {
	case agBuf = <-hlp.AgreementChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("should have received an Agreement message")
	}

	// Ensure we get a regeneration message
	select {
	case <-hlp.RegenChan:
		// Success
	case <-time.After(time.Second * 2):
		t.Fatal("no regeneration message received after halting")
	}

	// Retrieve Agreement
	ag := agreement.New(header.Header{})
	err := agreement.Unmarshal(&agBuf, ag)
	assert.NoError(t, err)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, ag.VotesPerStep[0], 1, 1))
	assert.NoError(t, hlp.Verify(hash, ag.VotesPerStep[1], 1, 2))
}

func TestSecondStepAfterFailure(t *testing.T) {
	hlp, hash := startReductionTest()

	// Start the first step
	if err := hlp.StartReduction(nil); err != nil {
		t.Fatal(err)
	}

	// Send events
	hlp.SendBatch(hash, 1, 2)

	// Ensure we get a regeneration message
	select {
	case <-hlp.RegenChan:
		// Success
	case <-time.After(time.Second * 2):
		t.Fatal("no regeneration message received after halting")
	}

	// Make sure no agreement message is sent
	select {
	case <-hlp.AgreementChan:
		t.Fatal("not supposed to construct an agreement if the first StepVotes is nil")
	case <-time.After(time.Second * 1):
		// Success
	}
}

func TestSecondStepTimeOut(t *testing.T) {
	hlp, _ := startReductionTest()

	// Start the first step
	if err := hlp.StartReduction(nil); err != nil {
		t.Fatal(err)
	}

	// Ensure we get a regeneration message
	select {
	case <-hlp.RegenChan:
		// Success
	case <-time.After(time.Second * 2):
		t.Fatal("no regeneration message received after halting")
	}

	// Make sure no agreement message is sent
	select {
	case <-hlp.AgreementChan:
		t.Fatal("not supposed to construct an agreement if the first StepVotes is nil")
	case <-time.After(time.Second * 1):
		// Success
	}
}

func startReductionTest() (*Helper, []byte) {
	committeeSize := 50
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp := NewHelper(bus, rpcBus, committeeSize)
	hash, _ := crypto.RandEntropy(32)
	hlp.Initialize(consensus.MockRoundUpdate(1, hlp.P, nil))
	return hlp, hash
}
