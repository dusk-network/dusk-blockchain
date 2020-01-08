package candidate

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type candidateCollector struct {
	candidateChan chan Candidate
}

func initCandidateCollector(sub eventbus.Subscriber) <-chan Candidate {
	candidateChan := make(chan Candidate, 100)
	collector := &candidateCollector{candidateChan}
	eventbus.NewTopicListener(sub, collector, topics.Candidate, eventbus.ChannelType)
	return candidateChan
}

func (c *candidateCollector) Collect(b bytes.Buffer) error {
	// validate performs a simple integrity check with an
	// incoming block's hash
	if err := Validate(b); err != nil {
		return err
	}

	cm := NewCandidate()
	if err := Decode(&b, cm); err != nil {
		return err
	}

	c.candidateChan <- *cm
	return nil
}
