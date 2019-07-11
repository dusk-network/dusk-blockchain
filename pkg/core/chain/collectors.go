package chain

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	blockCollector struct {
		blockChan chan<- *block.Block
	}

	certificateCollector struct {
		certificateChan chan<- certMsg
	}

	certMsg struct {
		hash []byte
		cert *block.Certificate
	}
)

// Init a block collector compatible with topics.Block and topics.Candidate
func initBlockCollector(eventBus *wire.EventBus, topic string) chan *block.Block {
	blockChan := make(chan *block.Block, 1)
	collector := &blockCollector{blockChan}
	go wire.NewTopicListener(eventBus, collector, topic).Accept()
	return blockChan
}

func (b *blockCollector) Collect(message *bytes.Buffer) error {
	blk := block.NewBlock()
	if err := blk.Decode(message); err != nil {
		return err
	}

	b.blockChan <- blk
	return nil
}

func initCertificateCollector(subscriber wire.EventSubscriber) <-chan certMsg {
	certificateChan := make(chan certMsg, 10)
	collector := &certificateCollector{certificateChan}
	go wire.NewTopicListener(subscriber, collector, string(topics.Certificate)).Accept()
	return certificateChan
}

func (c *certificateCollector) Collect(m *bytes.Buffer) error {
	var hash []byte
	if err := encoding.Read256(m, &hash); err != nil {
		return err
	}

	cert := block.EmptyCertificate()
	if err := cert.Decode(m); err != nil {
		return err
	}

	c.certificateChan <- certMsg{hash, cert}
	return nil
}
