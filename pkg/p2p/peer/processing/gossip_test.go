package processing

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	crypto "github.com/dusk-network/dusk-crypto/hash"
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

var Res []byte

func BenchmarkWriteReset(b *testing.B) {

	var m *bytes.Buffer
	hw := headerWriter{
		magicBuf: writeMagic(protocol.DevNet),
	}
	buf, _ := crypto.RandEntropy(1200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m = bytes.NewBuffer(buf)
		b.StartTimer()
		_ = hw.Write(m)
	}

	b.StopTimer()
	Res = m.Bytes()
}
