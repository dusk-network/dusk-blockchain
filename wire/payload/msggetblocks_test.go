package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
)

func TestMsgGetBlocksEncodeDecode(t *testing.T) {
	locator, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	stop, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	msg := NewMsgGetBlocks(locator, stop)
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgGetBlocks{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}
