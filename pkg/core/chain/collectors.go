package chain

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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
	eventbus.NewTopicListener(subscriber, collector, topics.Certificate, eventbus.ChannelType)
	return certificateChan
}

func (c *certificateCollector) Collect(m bytes.Buffer) error {
	hash := make([]byte, 32)
	if err := encoding.Read256(&m, hash); err != nil {
		return err
	}

	cert := block.EmptyCertificate()
	if err := marshalling.UnmarshalCertificate(&m, cert); err != nil {
		return err
	}

	c.certificateChan <- certMsg{hash, cert}
	return nil
}

func initHighestSeenCollector(sub eventbus.Subscriber) <-chan uint64 {
	highestSeenChan := make(chan uint64, 1)
	collector := &highestSeenCollector{highestSeenChan}
	eventbus.NewTopicListener(sub, collector, topics.HighestSeen, eventbus.ChannelType)
	return highestSeenChan
}

func (h *highestSeenCollector) Collect(m bytes.Buffer) error {
	var height uint64
	if err := encoding.ReadUint64LE(&m, &height); err != nil {
		return err
	}

	h.highestSeenChan <- height
	return nil
}
