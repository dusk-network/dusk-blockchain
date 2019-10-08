package protocol_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
)

var magicTest = []struct {
	magic   protocol.Magic
	payload []byte
}{
	{protocol.TestNet, []byte("pippo")},
	{protocol.DevNet, []byte("paperino")},
	{protocol.TestNet, []byte("pluto")},
	{protocol.MainNet, []byte("bruto")},
}

func TestExtract(t *testing.T) {
	var buf bytes.Buffer

	for _, tt := range magicTest {
		buf = tt.magic.ToBuffer()
		buf.ReadFrom(bytes.NewBuffer(tt.payload))
		magic, err := protocol.Extract(&buf)
		assert.Equal(t, magic, tt.magic)
		assert.NoError(t, err)
		assert.Equal(t, tt.payload, buf.Bytes())
	}
}
