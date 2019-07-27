package selection_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

// Test both the Marshal and Unmarshal functions, making sure the data is correctly
// stored to/retrieved from a Buffer.
func TestUnMarshal(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	se := selection.MockSelectionEvent(26, hash)

	buf := new(bytes.Buffer)
	assert.NoError(t, selection.MarshalScoreEvent(buf, se))

	other := &selection.ScoreEvent{Certificate: block.EmptyCertificate()}
	assert.NoError(t, selection.UnmarshalScoreEvent(buf, other))
	assert.True(t, other.Equal(se))
}
