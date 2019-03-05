package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

func TestMsgNotFoundEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	msg := payload.NewMsgNotFound()
	vec1 := payload.InvVect{
		Type: payload.InvTx,
		Hash: byte32,
	}

	vec2 := payload.InvVect{
		Type: payload.InvBlock,
		Hash: byte32,
	}

	msg.AddVector(vec1)
	msg.AddVector(vec2)
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := payload.NewMsgNotFound()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
