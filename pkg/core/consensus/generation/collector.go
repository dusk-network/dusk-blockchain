package generation

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	hashCollector struct {
		hashChannel chan []byte
	}
)

func initWinningHashCollector(subscriber wire.EventSubscriber) chan []byte {
	hashChannel := make(chan []byte, 1)
	collector := &hashCollector{hashChannel}
	go wire.NewTopicListener(subscriber, collector, msg.WinningBlockHashTopic).Accept()
	return hashChannel
}

func (c *hashCollector) Collect(message *bytes.Buffer) error {
	c.hashChannel <- message.Bytes()
	return nil
}
