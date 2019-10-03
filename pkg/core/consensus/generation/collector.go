package generation

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type (
	hashCollector struct {
		hashChannel chan []byte
	}
)

func initWinningHashCollector(subscriber eventbus.Subscriber) chan []byte {
	hashChannel := make(chan []byte, 1)
	collector := &hashCollector{hashChannel}
	eventbus.NewTopicListener(subscriber, collector, msg.WinningBlockHashTopic, eventbus.ChannelType)
	return hashChannel
}

func (c *hashCollector) Collect(message bytes.Buffer) error {
	//XXX: why do we callback into a channel?
	c.hashChannel <- message.Bytes()
	return nil
}
