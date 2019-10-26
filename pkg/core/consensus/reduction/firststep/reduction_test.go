package firststep

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestFirstStep(t *testing.T) {
	hlp, hash := startReductionTest()

	// Send events
	hlp.SendBatch(hash, 1, 1)

	// Wait for resulting StepVotes
	var svBuf bytes.Buffer
	select {
	case svBuf = <-hlp.StepVotesChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("should have received a StepVotes message")
	}
	// Retrieve StepVotes
	sv, err := agreement.UnmarshalStepVotes(&svBuf)
	assert.NoError(t, err)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, sv))
}

func TestFirstStepTimeOut(t *testing.T) {
	hlp, _ := startReductionTest()

	// No sending events here, as we want the step to timeout

	// Wait for resulting StepVotes
	var svBuf bytes.Buffer
	select {
	case svBuf = <-hlp.StepVotesChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("should have received a StepVotes message")
	}
	// Retrieve StepVotes
	_, err := agreement.UnmarshalStepVotes(&svBuf)
	// Should get an EOF
	assert.Error(t, err)
}

// Common functionality to start all reduction tests
func startReductionTest() (*Helper, []byte) {
	committeeSize := 50
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp := NewHelper(bus, rpcBus, &mockPlayer{}, &mockSigner{bus}, committeeSize)
	hash, _ := crypto.RandEntropy(32)
	hlp.Initialize(consensus.MockRoundUpdate(1, hlp.P, nil))
	hlp.StartReduction(hash)
	return hlp, hash
}

// No-op implementation of consensus.EventPlayer
type mockPlayer struct{}

func (m *mockPlayer) Resume(uint32) {}
func (m *mockPlayer) Pause(uint32)  {}
func (m *mockPlayer) Forward()      {}

type mockSigner struct {
	bus *eventbus.EventBus
}

func (m *mockSigner) Sign([]byte, []byte) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) SendAuthenticated(topics.Topic, []byte, *bytes.Buffer) error { return nil }
func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	m.bus.Publish(topic, b)
	return nil
}
