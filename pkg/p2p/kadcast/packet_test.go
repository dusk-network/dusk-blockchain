package kadcast

import (
	"bytes"
	"testing"
)

func TestPacketMarshalling(t *testing.T) {

	buf := make([]byte, 24+10)
	var i byte
	for ; i < byte(len(buf)); i++ {
		buf[i] = i
	}

	var p Packet
	unmarshalPacket(buf, &p)

	data := marshalPacket(p)

	if !bytes.Equal(data, buf) {
		t.Error("packet marshaling failed")
	}
}
