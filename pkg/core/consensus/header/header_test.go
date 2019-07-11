package header_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// Test that the MarshalSignableVote and UnmarshalSignableVote functions store/retrieve
// the passed data properly.
func TestSignableVote(t *testing.T) {
	red := &header.Header{}
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	test := &header.Header{}
	buf := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(buf, red))
	assert.NoError(t, header.UnmarshalSignableVote(buf, test))
	assert.Equal(t, red, test)
}

// This test ensures that both the Marshal and Unmarshal functions work as expected.
// The data which is stored and retrieved by both functions should remain the same.
func TestUnMarshal(t *testing.T) {
	unMarshaller := header.NewUnMarshaller()
	header1 := &header.Header{}
	header1.PubKeyBLS, _ = crypto.RandEntropy(129)
	header1.Round = uint64(5)
	header1.Step = uint8(2)
	header1.BlockHash, _ = crypto.RandEntropy(32)

	buf := new(bytes.Buffer)
	assert.NoError(t, unMarshaller.Marshal(buf, header1))
	header2 := &header.Header{}
	assert.NoError(t, unMarshaller.Unmarshal(buf, header2))

	assert.True(t, header1.Equal(header2))
}
