// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// TODO: implement light node registry.
var routingRegistry = map[protocol.ServiceFlag]map[topics.Topic]struct{}{
	// Full node
	protocol.FullNode: {
		topics.Tx:            {},
		topics.Candidate:     {},
		topics.NewBlock:      {},
		topics.Reduction:     {},
		topics.Agreement:     {},
		topics.AggrAgreement: {},
		topics.Ping:          {},
		topics.Pong:          {},
		topics.GetData:       {},
		topics.GetBlocks:     {},
		topics.Block:         {},
		topics.MemPool:       {},
		topics.Inv:           {},
		topics.GetCandidate:  {},
		topics.Addr:          {},
		topics.Challenge:     {},
		topics.Response:      {},
		topics.GetAddrs:      {},
	},
}

func canRoute(services protocol.ServiceFlag, topic topics.Topic) bool {
	_, ok := routingRegistry[services][topic]
	return ok
}
