package candidate

import (
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type candidateCollector struct {
	candidateChan chan message.Candidate
}

func initCandidateCollector(sub eventbus.Subscriber) <-chan message.Candidate {
	candidateChan := make(chan message.Candidate, 100)
	collector := &candidateCollector{candidateChan}
	collectListener := eventbus.NewCallbackListener(collector.Collect)
	if config.Get().General.SafeCallbackListener {
		collectListener = eventbus.NewSafeCallbackListener(collector.Collect)
	}
	sub.Subscribe(topics.Candidate, collectListener)
	return candidateChan
}

func (c *candidateCollector) Collect(m message.Message) error {
	// validate performs a simple integrity check with an
	// incoming block's hash
	cm := m.Payload().(message.Candidate)
	if err := ValidateCandidate(cm); err != nil {
		return err
	}

	c.candidateChan <- cm
	return nil
}
