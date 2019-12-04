package processing

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

func TestWriteReadFrame(t *testing.T) {
	b := bytes.NewBufferString("pippo")
	digest := sha3.Sum256(b.Bytes())
	WriteFrame(b, protocol.TestNet, digest[0:ChecksumLength])

	length, _ := ReadFrame(b)
	buf := make([]byte, length)
	b.Read(buf)

	// Remove magic and checksum bytes
	buf = buf[8:]

	assert.Equal(t, "pippo", string(buf))
}
