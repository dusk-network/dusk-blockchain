package wire_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
)

func TestAddTopic(t *testing.T) {
	buf := bytes.NewBufferString("This is a test")
	topic := topics.Gossip
	newBuffer, err := wire.AddTopic(buf, topic)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x67, 0x6f, 0x73, 0x73, 0x69, 0x70, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x74, 0x65, 0x73, 0x74}, newBuffer.Bytes())
}
