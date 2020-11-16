package protocol

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

func TestWriteReadFrame(t *testing.T) {
	b := bytes.NewBufferString("pippo")
	digest := sha3.Sum256(b.Bytes())
	WriteFrame(b, TestNet, digest[0:checksum.Length])

	length, _ := ReadFrame(b)
	buf := make([]byte, length)
	b.Read(buf)

	// Remove magic and checksum bytes
	buf = buf[8:]

	assert.Equal(t, "pippo", string(buf))
}
