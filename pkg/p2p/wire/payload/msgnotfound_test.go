package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgNotFoundEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	msg := NewMsgNotFound()
	vec1 := InvVect{
		Type: InvTx,
		Hash: byte32,
	}

	vec2 := InvVect{
		Type: InvBlock,
		Hash: byte32,
	}

	msg.AddVector(vec1)
	msg.AddVector(vec2)
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := NewMsgNotFound()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
