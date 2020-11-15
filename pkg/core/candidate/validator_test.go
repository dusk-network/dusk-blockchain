package candidate

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/require"
)

// Ensure that the behavior of the validator works as intended.
// It should republish blocks with a correct hash and root.
func TestValidatorValidBlock(t *testing.T) {
	// Send it over to the validator
	cm := mockCandidate()
	msg := message.New(topics.Candidate, cm)
	assert.NoError(t, Validate(msg))
}

// Ensure that blocks with an invalid hash or tx root will not be
// republished.
func TestValidatorInvalidBlock(t *testing.T) {
	// preventing unnecessary logging on expected errors
	logrus.SetLevel(logrus.FatalLevel)
	cm := mockCandidate()
	// Remove one of the transactions to remove the integrity of
	// the merkle root
	cm.Block.Txs = cm.Block.Txs[1:]
	msg := message.New(topics.Candidate, cm)
	assert.Error(t, Validate(msg))
}
