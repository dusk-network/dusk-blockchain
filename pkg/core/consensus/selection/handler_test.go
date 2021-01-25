// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package selection

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

func TestPriority(t *testing.T) {
	// mock candidate
	genesis := config.DecodeGenesis()

	hdr := header.Mock()
	ev := message.MockScore(hdr, *genesis)

	// Comparing the same event should return true
	handler := NewScoreHandler(nil)
	assert.True(t, handler.Priority(ev, ev))
}
