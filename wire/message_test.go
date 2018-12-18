package wire

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/protocol"
)

func TestWriteReadMessage(t *testing.T) {
	addr1 := payload.NewNetAddress("202.108.250.180", 9999)
	addr2 := payload.NewNetAddress("224.164.2.18", 9999)

	msg := payload.NewMsgVersion(protocol.ProtocolVersion, addr1, addr2)
	buf := new(bytes.Buffer)
	if err := WriteMessage(buf, protocol.DevNet, msg); err != nil {
		t.Fatal(err)
	}

	msg2, err := ReadMessage(buf, protocol.DevNet)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}

func TestWriteReadMessageNoPayload(t *testing.T) {
	msg := payload.NewMsgVerAck()
	bs := make([]byte, 0, HeaderSize)
	buf := bytes.NewBuffer(bs)
	if err := WriteMessage(buf, protocol.DevNet, msg); err != nil {
		t.Fatal(err)
	}

	msg2, err := ReadMessage(buf, protocol.DevNet)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
