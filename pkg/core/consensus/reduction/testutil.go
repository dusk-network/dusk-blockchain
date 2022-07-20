// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package reduction

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// PrepareSendReductionTest tests that the reduction step completes without problems
// and produces a StepVotesMsg in case it receives enough valid Reduction messages.
func PrepareSendReductionTest(hlp *Helper, stepFn consensus.PhaseFn) func(t *testing.T) {
	return func(t *testing.T) {
		require := require.New(t)

		streamer := eventbus.NewGossipStreamer()
		l := eventbus.NewStreamListener(streamer)
		hlp.EventBus.Subscribe(topics.Gossip, l)

		ctx, cancel := context.WithCancel(context.Background())
		go func(cancel context.CancelFunc) {
			_, err := streamer.Read()
			require.NoError(err)
			require.Equal(streamer.SeenTopics()[0], topics.Reduction)
			cancel()
		}(cancel)

		evChan := make(chan message.Message, 1)
		n := stepFn.Run(ctx, consensus.NewQueue(), evChan, consensus.MockRoundUpdate(uint64(1), hlp.P), uint8(2))
		require.Nil(n)
	}
}

// Helper for reducing test boilerplate.
type Helper struct {
	*consensus.Emitter
	lock               sync.RWMutex
	failOnVerification bool

	ThisSender       []byte
	ProvisionersKeys []key.Keys
	P                *user.Provisioners
	Nr               int
	Handler          *Handler
}

// NewHelper creates a Helper.
func NewHelper(provisioners int, timeOut time.Duration) *Helper {
	p, provisionersKeys := consensus.MockProvisioners(provisioners)

	emitter := consensus.MockEmitter(timeOut)
	emitter.Keys = provisionersKeys[0]
	seed := []byte{0, 0, 0, 0}

	hlp := &Helper{
		failOnVerification: false,

		ThisSender:       emitter.Keys.BLSPubKey,
		ProvisionersKeys: provisionersKeys,
		P:                p,
		Nr:               provisioners,
		Handler:          NewHandler(emitter.Keys, *p, seed),
		Emitter:          emitter,
	}

	return hlp
}

// Verify StepVotes. The step must be specified otherwise verification would be dependent on the state of the Helper.
func (hlp *Helper) Verify(hash []byte, sv message.StepVotes, round uint64, step uint8) error {
	seed := []byte{0, 0, 0, 0}

	vc := hlp.P.CreateVotingCommittee(seed, round, step, hlp.Nr)
	sub := vc.IntersectCluster(sv.BitSet)

	apk, err := agreement.AggregatePks(hlp.P, sub.Set)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, hash, apk, sv.Signature)
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus.
func (hlp *Helper) Spawn(hash []byte, round uint64, step uint8) []message.Reduction {
	evs := make([]message.Reduction, 0, hlp.Nr)
	i := 0

	for count := 0; count < hlp.Handler.Quorum(round); {
		ev := message.MockReduction(hash, round, step, hlp.ProvisionersKeys, i)
		evs = append(evs, ev)

		i++
		count += hlp.Handler.VotesFor(hlp.ProvisionersKeys[i].BLSPubKey, round, step)
	}

	return evs
}

// FailOnVerification tells the RPC bus to return an error.
func (hlp *Helper) FailOnVerification(flag bool) {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.failOnVerification = flag
}

func (hlp *Helper) shouldFailVerification() bool {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()

	f := hlp.failOnVerification
	return f
}

// ProcessCandidateVerificationRequest is a callback used by the firststep
// reduction to verify potential winning candidates.
func (hlp *Helper) ProcessCandidateVerificationRequest(ctx context.Context, blk block.Block) error {
	if hlp.shouldFailVerification() {
		return errors.New("verification failed")
	}

	return nil
}
