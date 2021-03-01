// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// CandidateBroker holds instances to RPCBus and responseChan.
type CandidateBroker struct {
	db database.DB
}

// NewCandidateBroker will create new CandidateBroker.
func NewCandidateBroker(db database.DB) *CandidateBroker {
	return &CandidateBroker{db}
}

// ProvideCandidate for a given (m *bytes.Buffer).
func (c *CandidateBroker) ProvideCandidate(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	msg := m.Payload().(message.GetCandidate)

	var cm block.Block

	if err := c.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(msg.Hash)
		return err
	}); err != nil {
		return nil, err
	}

	candidateBytes := new(bytes.Buffer)
	if err := message.MarshalBlock(candidateBytes, &cm); err != nil {
		return nil, err
	}

	if err := topics.Prepend(candidateBytes, topics.Candidate); err != nil {
		return nil, err
	}

	return []bytes.Buffer{*candidateBytes}, nil
}
