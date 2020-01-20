package chain

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
)

type (
	certificateCollector struct {
		certificateChan chan<- certMsg
	}

	certMsg struct {
		hash []byte
		cert *block.Certificate
	}

	highestSeenCollector struct {
		highestSeenChan chan<- uint64
	}
)

func initCertificateCollector(subscriber eventbus.Subscriber) <-chan certMsg {
	certificateChan := make(chan certMsg, 10)
	collector := &certificateCollector{certificateChan}
	l := eventbus.NewCallbackListener(collector.Collect)
	subscriber.Subscribe(topics.Certificate, l)
	return certificateChan
}

func (c *certificateCollector) Collect(m message.Message) error {
	aggro := m.Payload().(message.Agreement)
	cert := aggro.GenerateCertificate()
	c.certificateChan <- certMsg{aggro.State().BlockHash, cert}
	return nil
}

func initHighestSeenCollector(sub eventbus.Subscriber) <-chan uint64 {
	highestSeenChan := make(chan uint64, 1)
	collector := &highestSeenCollector{highestSeenChan}
	l := eventbus.NewCallbackListener(collector.Collect)
	sub.Subscribe(topics.HighestSeen, l)
	return highestSeenChan
}

func (h *highestSeenCollector) Collect(m message.Message) error {
	height := m.Payload().(uint64)
	h.highestSeenChan <- height
	return nil
}
