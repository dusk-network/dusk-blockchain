package chain

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type (
	certificateCollector struct {
		certificateChan chan<- certMsg
	}

	certMsg struct {
		hash      []byte
		cert      *block.Certificate
		committee [][]byte
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
	cmsg := m.Payload().(message.Certificate)
	cert := cmsg.Ag.GenerateCertificate()
	c.certificateChan <- certMsg{cmsg.Ag.State().BlockHash, cert, cmsg.Keys}
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
	height, _ := message.ConvU64(m.Payload())
	h.highestSeenChan <- height
	return nil
}
