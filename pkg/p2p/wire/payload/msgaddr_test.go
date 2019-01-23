package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

func TestMsgAddrEncodeDecode(t *testing.T) {
	msg := payload.NewMsgAddr()

	addr1 := payload.NewNetAddress("224.176.128.1", 9999)
	addr2 := payload.NewNetAddress("224.164.2.18", 9999)
	addr3 := payload.NewNetAddress("202.108.250.180", 9999)

	msg.AddAddr(addr1)
	msg.AddAddr(addr2)
	msg.AddAddr(addr3)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := payload.NewMsgAddr()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
