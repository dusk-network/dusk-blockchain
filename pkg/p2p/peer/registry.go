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

// TODO: implement light node registry
var routingRegistry = map[protocol.ServiceFlag]map[topics.Topic]struct{}{
	// Full node
	protocol.FullNode: map[topics.Topic]struct{}{
		topics.Tx:           struct{}{},
		topics.Candidate:    struct{}{},
		topics.Score:        struct{}{},
		topics.Reduction:    struct{}{},
		topics.Agreement:    struct{}{},
		topics.Ping:         struct{}{},
		topics.Pong:         struct{}{},
		topics.GetData:      struct{}{},
		topics.GetBlocks:    struct{}{},
		topics.Block:        struct{}{},
		topics.MemPool:      struct{}{},
		topics.Inv:          struct{}{},
		topics.GetCandidate: struct{}{},
	},
	// Voucher node
	// TODO: add message types
	protocol.VoucherNode: map[topics.Topic]struct{}{},
}

func canRoute(services protocol.ServiceFlag, topic topics.Topic) bool {
	_, ok := routingRegistry[services][topic]
	return ok
}
