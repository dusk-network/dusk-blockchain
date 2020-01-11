package message_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Test both the Marshal and Unmarshal functions, making sure the data is correctly
// stored to/retrieved from a Buffer.
func TestUnMarshal(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	se := selection.MockSelectionEvent(hash)

	buf := new(bytes.Buffer)
	assert.NoError(t, message.MarshalScore(buf, se))

	other := &message.Score{}
	assert.NoError(t, message.UnmarshalScore(buf, other))
	assert.True(t, other.Equal(*se))
}
