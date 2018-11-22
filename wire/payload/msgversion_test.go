package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgVersionEncodeDecode(t *testing.T) {
	// TODO: add actual I2P address
	msg := NewMsgVersion("placeholder")

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgVersion{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}
