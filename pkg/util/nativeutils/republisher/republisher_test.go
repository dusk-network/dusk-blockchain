package republisher_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/stretchr/testify/assert"
)

func TestRepublisher(t *testing.T) {
	eb := eventbus.New()
	gossipChan := make(chan bytes.Buffer, 1)
	gl := eventbus.NewChanListener(gossipChan)
	eb.Subscribe(topics.Gossip, gl)

	republisher.New(eb, topics.Agreement)

	mockAggro := bytes.NewBuffer([]byte{1})
	eb.Publish(topics.Agreement, mockAggro)

	packet := <-gossipChan
	tpc, err := topics.Extract(&packet)
	assert.NoError(t, err)
	assert.Equal(t, topics.Agreement, tpc)
	assert.Equal(t, []byte{1}, packet.Bytes())
}
