// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package selection_test

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type tparm struct {
	bg   blockgenerator.BlockGenerator
	msgs []message.Score
}

func TestSelection(t *testing.T) {
	hlp := selection.NewHelper(10)
	consensusTimeOut := 300 * time.Millisecond
	table := map[string]tparm{
		"ExternalWinningScore": {
			bg:   blockgenerator.Mock(hlp.Emitter, true),
			msgs: hlp.Spawn(),
		},

		"InternalWinningScore": {
			bg:   blockgenerator.Mock(hlp.Emitter, false),
			msgs: []message.Score{},
		},
	}
	_, db := lite.CreateDBConnection()

	for name, ttest := range table {
		t.Run(name, func(t *testing.T) {
			ttestCB := func(require *require.Assertions, p consensus.InternalPacket, _ *eventbus.GossipStreamer) {
				require.NotNil(p)
				messageScore := p.(message.Score)
				require.NotEmpty(messageScore)
			}

			testPhase := consensus.NewTestPhase(t, ttestCB, nil)
			sel := selection.New(testPhase, ttest.bg, hlp.Emitter, consensusTimeOut, db)
			selFn := sel.Initialize(nil)

			msgChan := make(chan message.Message, 1)
			msgs := ttest.msgs
			go func(msgs []message.Score) {
				for _, msg := range msgs {
					msgChan <- message.New(topics.Score, msg)
				}
			}(msgs)
			testCallbackPhase := selFn.Run(context.Background(), consensus.NewQueue(), msgChan, hlp.RoundUpdate(), hlp.Step)
			_ = testCallbackPhase.Run(context.Background(), nil, nil, hlp.RoundUpdate(), hlp.Step+1)
		})
	}
}
