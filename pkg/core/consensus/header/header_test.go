package header

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// Test that the MarshalSignableVote and UnmarshalSignableVote functions store/retrieve
// the passed data properly.
func TestSignableVote(t *testing.T) {
	red := &Header{}
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	test := &Header{}
	buf := new(bytes.Buffer)
	assert.NoError(t, MarshalSignableVote(buf, red))
	assert.NoError(t, UnmarshalSignableVote(buf, test))
	assert.Equal(t, red, test)
}
