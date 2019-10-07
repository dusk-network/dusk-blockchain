package processing

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteReadFrame(t *testing.T) {
	b := bytes.NewBufferString("pippo")
	WriteFrame(b)
	fmt.Println(b.Bytes())

	buf, _ := ReadFrame(b)
	assert.Equal(t, "pippo", string(buf))
}
