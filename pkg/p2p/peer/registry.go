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
	1: map[topics.Topic]struct{}{
		topics.Tx:           struct{}{},
		topics.Candidate:    struct{}{},
		topics.Score:        struct{}{},
		topics.Reduction:    struct{}{},
		topics.Agreement:    struct{}{},
		topics.GetCandidate: struct{}{},
	},
	// Voucher node
	// TODO: add message types
	3: map[topics.Topic]struct{}{},
}
