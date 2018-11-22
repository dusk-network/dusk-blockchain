package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgAddrEncodeDecode(t *testing.T) {
	msg := NewMsgAddr()

	// TODO: add actual I2P addresses
	msg.AddAddr("placeholder")
	msg.AddAddr("placeholder2")
	msg.AddAddr("placeholder3")

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := NewMsgAddr()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
