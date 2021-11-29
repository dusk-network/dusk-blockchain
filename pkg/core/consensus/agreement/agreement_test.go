// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// TestMockValidity ensures that we don't go into a wild goose chase if our
// mock system gets screwed up.
func TestMockValidity(t *testing.T) {
	nr := 10
	hlp := agreement.NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := agreement.NewHandler(hlp.Keys, *hlp.P, []byte{0, 0, 0, 0})

	evs := hlp.Spawn(hash)
	for _, ev := range evs {
		if !assert.NoError(t, handler.Verify(ev)) {
			t.FailNow()
		}
	}
}

// Test the accumulation of agreement events. It should result in the agreement
// component sending a valid certificate.
func TestAgreement(t *testing.T) {
	nr := 10
	hlp := agreement.NewHelper(nr)
	blk := helper.RandomBlock(1, 1)
	_, db := lite.CreateDBConnection()

	assert.NoError(t, db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(*blk)
	}))

	loop := agreement.New(hlp.Emitter, db, nil)

	agreementEvs := hlp.Spawn(blk.Header.Hash)
	agreementChan := make(chan message.Message, 100)
	aggrAgreementChan := make(chan message.Message, 100)

	for _, aggro := range agreementEvs {
		agreementChan <- message.New(topics.Agreement, aggro)
	}

	ctx := context.Background()
	results := loop.Run(ctx, consensus.NewQueue(), agreementChan, aggrAgreementChan, hlp.RoundUpdate(blk.Header.Hash))

	assert.Equal(t, blk.Header.Hash, results.Blk.Header.Hash)
}

// Test the usage of the candidate requestor in case of a missing candidate block.
func TestRequestor(t *testing.T) {
	nr := 10
	hlp := agreement.NewHelper(nr)
	blk := helper.RandomBlock(1, 1)
	_, db := lite.CreateDBConnection()

	req := candidate.NewRequestor(hlp.Emitter.EventBus)

	c := make(chan message.Message, 100)
	l := eventbus.NewChanListener(c)
	hlp.Emitter.EventBus.Subscribe(topics.Gossip, l)

	loop := agreement.New(hlp.Emitter, db, req)

	agreementEvs := hlp.Spawn(blk.Header.Hash)
	agreementChan := make(chan message.Message, 100)
	aggrAgreementChan := make(chan message.Message, 100)

	for _, aggro := range agreementEvs {
		agreementChan <- message.New(topics.Agreement, aggro)
	}

	ctx := context.Background()
	resChan := make(chan consensus.Results, 1)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		results := loop.Run(ctx, consensus.NewQueue(), agreementChan, aggrAgreementChan, hlp.RoundUpdate(blk.Header.Hash))

		wg.Done()
		resChan <- results
	}()

	// Catch request, and return the desired block.
	// We loop here because we also catch republished messages.
	for {
		m := <-c
		b := m.Payload().(message.SafeBuffer).Buffer

		request, err := message.Unmarshal(&b, nil)
		assert.NoError(t, err)

		if request.Category() == topics.GetCandidate {
			assert.Equal(t, request.Payload().(message.GetCandidate).Hash, blk.Header.Hash)

			_, err := req.ProcessCandidate("", message.New(topics.Candidate, *blk))
			assert.NoError(t, err)

			wg.Wait()

			results := <-resChan
			assert.NoError(t, results.Err)
			assert.Equal(t, blk.Header.Hash, results.Blk.Header.Hash)
			return
		}
	}
}
