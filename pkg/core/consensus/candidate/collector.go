package candidate

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type hashCollector struct {
	hashChan chan<- string
}

func initHashCollector(subscriber wire.EventSubscriber, topic string) <-chan string {
	hashChan := make(chan string, 1)
	collector := &hashCollector{hashChan}
	go wire.NewTopicListener(subscriber, collector, topic).Accept()
	return hashChan
}

func (h *hashCollector) Collect(m *bytes.Buffer) error {
	h.hashChan <- hex.EncodeToString(m.Bytes())
	return nil
}
