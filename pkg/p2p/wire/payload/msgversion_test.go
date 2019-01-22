package payload

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestMsgVersionEncodeDecode(t *testing.T) {
	pver := protocol.ProtocolVersion

	addr1 := NewNetAddress("202.108.250.180", 9999)
	addr2 := NewNetAddress("224.164.2.18", 9999)

	msg := NewMsgVersion(pver, addr1, addr2, rand.Uint64())

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgVersion{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}
