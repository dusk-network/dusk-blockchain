package agreement_test

import (
	"context"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// TestMockValidity ensures that we don't go into a wild goose chase if our
// mock system gets screwed up
func TestMockValidity(t *testing.T) {
	nr := 50
	hlp := agreement.NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := agreement.NewHandler(hlp.Keys, *hlp.P)

	evs := hlp.Spawn(hash)
	for _, ev := range evs {
		if !assert.NoError(t, handler.Verify(ev)) {
			t.FailNow()
		}
	}
}

// Test the accumulation of agreement events. It should result in the agreement
// component sending a valid certificate
func TestAgreement(t *testing.T) {
	nr := 50
	hlp := agreement.NewHelper(nr)
	blk := helper.RandomBlock(1, 1)
	_, db := lite.CreateDBConnection()

	assert.NoError(t, db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(*blk)
	}))

	loop := agreement.New(hlp.Emitter, db, make(chan consensus.Results, 1))

	agreementEvs := hlp.Spawn(blk.Header.Hash)
	agreementChan := make(chan message.Message, 100)

	for _, aggro := range agreementEvs {
		agreementChan <- message.New(topics.Agreement, aggro)
	}

	ctx := context.Background()
	results := loop.Run(ctx, consensus.NewQueue(), agreementChan, hlp.RoundUpdate(blk.Header.Hash))

	assert.Equal(t, blk.Header.Hash, results.Blk.Header.Hash)
}
