package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgAddrEncodeDecode(t *testing.T) {
	msg := NewMsgAddr()

	addr1 := NewNetAddress("224.176.128.1", 9999)
	addr2 := NewNetAddress("224.164.2.18", 9999)
	addr3 := NewNetAddress("202.108.250.180", 9999)

	msg.AddAddr(addr1)
	msg.AddAddr(addr2)
	msg.AddAddr(addr3)

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
