package wire

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/wire/payload"
)

func TestWriteReadMessage(t *testing.T) {
	// TODO: add actual I2P address
	msg := payload.NewMsgVersion("placeholder", ProtocolVersion)
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
