package wire

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/payload"
)

func TestWriteReadMessage(t *testing.T) {
	// TODO: add actual I2P address
	msg := payload.NewMsgVersion("placeholder")
	buf := new(bytes.Buffer)
	if err := WriteMessage(buf, 0x91919191, msg); err != nil {
		t.Fatal(err)
	}

	msg2, err := ReadMessage(buf, 0x91919191)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}

func TestWriteReadMessageNoPayload(t *testing.T) {
	msg := payload.NewMsgVerAck()
	bs := make([]byte, 0, HeaderSize)
	buf := bytes.NewBuffer(bs)
	if err := WriteMessage(buf, 0x91919191, msg); err != nil {
		t.Fatal(err)
	}

	msg2, err := ReadMessage(buf, 0x91919191)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
