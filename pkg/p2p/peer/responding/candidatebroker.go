package responding

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// CandidateBroker holds instances to RPCBus and responseChan
type CandidateBroker struct {
	db database.DB
}

// NewCandidateBroker will create new CandidateBroker
func NewCandidateBroker(db database.DB) *CandidateBroker {
	return &CandidateBroker{db}
}

// ProvideCandidate for a given (m *bytes.Buffer)
func (c *CandidateBroker) ProvideCandidate(m message.Message) ([]*bytes.Buffer, error) {
	msg := m.Payload().(message.GetCandidate)
	var cm message.Candidate
	if err := c.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(msg.Hash)
		return err
	}); err != nil {
		return nil, err
	}

	candidateBytes := new(bytes.Buffer)
	if err := message.MarshalCandidate(candidateBytes, cm); err != nil {
		return nil, err
	}

	if err := topics.Prepend(candidateBytes, topics.Candidate); err != nil {
		return nil, err
	}

	return []*bytes.Buffer{candidateBytes}, nil
}
