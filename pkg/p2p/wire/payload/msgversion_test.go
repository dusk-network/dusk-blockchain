package payload_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestMsgVersionEncodeDecode(t *testing.T) {
	pver := protocol.NodeVer

	addr1 := payload.NewNetAddress("202.108.250.180", 9999)
	addr2 := payload.NewNetAddress("224.164.2.18", 9999)

	services := protocol.FullNode

	msg := payload.NewMsgVersion(pver, addr1, addr2, services, rand.Uint64())

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgVersion{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}
