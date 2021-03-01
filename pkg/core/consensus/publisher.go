// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Publisher is used to direct consensus messages from the peer.MessageProcessor
// to the consensus components.
type Publisher struct {
	publisher eventbus.Publisher
}

// NewPublisher returns an initialized Publisher.
func NewPublisher(publisher eventbus.Publisher) *Publisher {
	return &Publisher{publisher}
}

// Process incoming consensus messages.
// Satisfies the peer.ProcessorFunc interface.
func (p *Publisher) Process(srcPeerID string, msg message.Message) ([]bytes.Buffer, error) {
	p.publisher.Publish(msg.Category(), msg)
	return nil, nil
}
