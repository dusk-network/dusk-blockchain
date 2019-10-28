package selection_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Test both the Marshal and Unmarshal functions, making sure the data is correctly
// stored to/retrieved from a Buffer.
func TestUnMarshal(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	se := selection.MockSelectionEvent(hash, consensus.MockBidList(2))

	buf := new(bytes.Buffer)
	assert.NoError(t, selection.MarshalScore(buf, se))

	other := &selection.Score{}
	assert.NoError(t, selection.UnmarshalScore(buf, other))
	assert.True(t, other.Equal(se))
}
