package wire

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/wire/payload"
)

func TestWriteReadMessage(t *testing.T) {
	addr1 := payload.NewNetAddress("202.108.250.180", 9999)
	addr2 := payload.NewNetAddress("224.164.2.18", 9999)

	msg := payload.NewMsgVersion(ProtocolVersion, addr1, addr2)
	buf := new(bytes.Buffer)
	if err := WriteMessage(buf, DevNet, msg); err != nil {
		t.Fatal(err)
	}

	_, msg2, err := ReadMessage(buf, DevNet)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}

func TestWriteReadMessageNoPayload(t *testing.T) {
	msg := payload.NewMsgVerAck()
	bs := make([]byte, 0, HeaderSize)
	buf := bytes.NewBuffer(bs)
	if err := WriteMessage(buf, DevNet, msg); err != nil {
		t.Fatal(err)
	}

	_, msg2, err := ReadMessage(buf, DevNet)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
