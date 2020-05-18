package republisher_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/stretchr/testify/assert"
)

// TestRepublisher tests that a republished message on the Gossip channel is a
// byte array containing the topic in encoded form
func TestRepublisher(t *testing.T) {
	eb := eventbus.New()
	gossipChan := make(chan message.Message, 1)
	gl := eventbus.NewChanListener(gossipChan)
	eb.Subscribe(topics.Gossip, gl)

	republisher.New(eb, topics.Agreement)

	mockAggro := bytes.NewBuffer([]byte{1})
	msg := message.New(topics.Agreement, *mockAggro)
	eb.Publish(topics.Agreement, msg)

	mPack := <-gossipChan
	packet := mPack.Payload().(message.SafeBuffer)

	tpc, err := topics.Extract(&packet)
	assert.NoError(t, err)
	assert.Equal(t, topics.Agreement, tpc)
	assert.Equal(t, []byte{1}, packet.Bytes())
}
