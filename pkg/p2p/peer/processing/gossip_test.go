package processing_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	g := processing.NewGossip(protocol.DevNet)

	m := bytes.NewBufferString("pippo")

	if !assert.NoError(t, g.Process(m)) {
		assert.FailNow(t, "error in processing buffer")
	}

	length, err := processing.ReadFrame(m)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "error in reading frame")
	}

	msg := make([]byte, length)
	if _, err := m.Read(msg); err != nil {
		assert.FailNow(t, fmt.Sprintf("error in reading the message with length %d", length))
	}

	buf := protocol.DevNet.ToBuffer()
	buf.Write([]byte("pippo"))
	assert.Equal(t, buf.Bytes(), msg)
}

func TestUnpackLength(t *testing.T) {
	test := "pippo"
	b := bytes.NewBufferString(test)

	g := processing.NewGossip(protocol.DevNet)
	assert.NoError(t, g.Process(b))

	length, err := g.UnpackLength(b)
	assert.NoError(t, err)

	assert.Equal(t, len(test), int(length))
}
