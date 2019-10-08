package processing

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteReadFrame(t *testing.T) {
	b := bytes.NewBufferString("pippo")
	WriteFrame(b)

	length, _ := ReadFrame(b)
	buf := make([]byte, length)
	b.Read(buf)

	assert.Equal(t, "pippo", string(buf))
}
