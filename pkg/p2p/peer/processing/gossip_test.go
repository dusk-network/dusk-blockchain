package processing

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	g := NewGossip(protocol.DevNet)

	m := bytes.NewBufferString("pippo")

	if !assert.NoError(t, g.Process(m)) {
		assert.FailNow(t, "error in processing buffer")
	}

	msg, err := ReadFrame(m)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "error in reading frame")
	}

	b := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(b, uint32(protocol.DevNet)); err != nil {
		b.Write([]byte("pippo"))
		assert.Equal(t, b.Bytes(), msg)
	}
}
