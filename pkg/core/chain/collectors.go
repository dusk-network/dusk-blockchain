package chain

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
)

type candidateCollector struct {
	blockChan chan<- *block.Block
}

func initCandidateCollector(eventBus *eventbus.EventBus) chan *block.Block {
	blockChan := make(chan *block.Block, 1)
	collector := &candidateCollector{blockChan}
	eventbus.NewTopicListener(eventBus, collector, topics.Candidate, eventbus.ChannelType)
	return blockChan
}

func (c *candidateCollector) Collect(message bytes.Buffer) error {
	hdr := header.Header{}
	if err := header.Unmarshal(&message, &hdr); err != nil {
		return err
	}

	blk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&message, blk); err != nil {
		return err
	}

	c.blockChan <- blk
	return nil
}
