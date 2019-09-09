package maintainer_test

import (
	"bytes"
	"math/rand"
	"os"
	"testing"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	litedb "github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/database"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/stretchr/testify/assert"
)

const dbPath = "testDb"
const pass = "password"

// Test that the maintainer will properly send new stake and bid transactions, when
// one is about to expire, or if none exist.
func TestMaintainStakesAndBids(t *testing.T) {
	// Initial setup
	bus := wire.NewEventBus()
	wdb, err := database.New(dbPath)
	defer os.RemoveAll(dbPath)

	w, err := wallet.New(rand.Read, 2, wdb, wallet.GenerateDecoys, wallet.GenerateInputs, pass)
	assert.NoError(t, err)
	defer os.Remove("wallet.dat")

	_, db := lite.CreateDBConnection()
	// Ensure we have a genesis block
	genesisBlock := cfg.DecodeGenesis()
	assert.NoError(t, db.Update(func(t litedb.Transaction) error {
		return t.StoreBlock(genesisBlock)
	}))

	k, err := w.ReconstructK()
	assert.NoError(t, err)
	assert.NoError(t, maintainer.Launch(bus, db, w.ConsensusKeys().BLSPubKeyBytes, zkproof.CalculateM(k), transactor.New(w, db), 10, 10, 5))

	// Send round update.
	// We have no stakes and no bids active. We should receive one of both
	txChan := make(chan *bytes.Buffer, 2)
	bus.Subscribe(string(topics.Tx), txChan)

	consensus.UpdateRound(bus, 1)
	txs := receiveTxs(t, txChan)

	// Check correctness
	// We get the bid first
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)

	// Send round update, to a point where the buffer will kick in
	consensus.UpdateRound(bus, 6)

	// We should get another set of two txs
	txs = receiveTxs(t, txChan)
	// Check correctness
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)
}

func receiveTxs(t *testing.T, txChan chan *bytes.Buffer) []transactions.Transaction {
	var txs []transactions.Transaction
	for i := 0; i < 2; i++ {
		txBuf := <-txChan
		tx, err := transactions.Unmarshal(txBuf)
		assert.NoError(t, err)
		txs = append(txs, tx)
	}

	return txs
}
