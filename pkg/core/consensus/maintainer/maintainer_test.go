package maintainer_test

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	litedb "github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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
	bus, txChan := setupMaintainerTest(t)
	defer os.RemoveAll(dbPath)
	defer os.Remove("wallet.dat")

	// receive first txs
	txs := receiveTxs(t, txChan)
	// Check correctness
	// We get the bid first
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)

	// Populate committee and bidlist
	propagateTxValues(txs, bus, 1)

	time.Sleep(200 * time.Millisecond)

	// Send round update, to update the end heights.
	consensus.UpdateRound(bus, 1)

	// Send round update, to a point where the buffer will kick in
	consensus.UpdateRound(bus, 6)

	// We should get another set of two txs
	txs = receiveTxs(t, txChan)
	// Check correctness
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {
	bus, txChan := setupMaintainerTest(t)
	defer os.RemoveAll(dbPath)
	defer os.Remove("wallet.dat")

	// receive first txs
	txs := receiveTxs(t, txChan)
	propagateTxValues(txs, bus, 1)

	// Update round. We should not receive any new txs
	consensus.UpdateRound(bus, 1)

	select {
	case <-txChan:
		t.Fatal("was not supposed to get another tx in txChan")
	case <-time.After(2 * time.Second):
		// success
	}
}

func propagateTxValues(txs []transactions.Transaction, bus *wire.EventBus, height uint64) {
	for _, tx := range txs {
		switch tx.Type() {
		case transactions.BidType:
			bid := tx.(*transactions.Bid)
			x := user.CalculateX(bid.Outputs[0].Commitment.Bytes(), bid.M)
			x.EndHeight = height + bid.Lock
			propagateBid(x, bus)
		case transactions.StakeType:
			stake := tx.(*transactions.Stake)
			propagateStake(stake, height, bus)
		}
	}
}

func propagateBid(bid user.Bid, bus *wire.EventBus) {
	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, bid.X[:]); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, bid.M[:]); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buf, bid.EndHeight); err != nil {
		panic(err)
	}

	bus.Publish(msg.BidListTopic, buf)
}

func propagateStake(tx *transactions.Stake, startHeight uint64, bus *wire.EventBus) {
	buffer := bytes.NewBuffer(tx.PubKeyEd)
	if err := encoding.WriteVarBytes(buffer, tx.PubKeyBLS); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buffer, tx.Outputs[0].EncryptedAmount.BigInt().Uint64()); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buffer, startHeight); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buffer, startHeight+tx.Lock); err != nil {
		panic(err)
	}

	bus.Publish(msg.NewProvisionerTopic, buffer)
}

func setupMaintainerTest(t *testing.T) (*wire.EventBus, chan *bytes.Buffer) {
	// Initial setup
	bus := wire.NewEventBus()
	wdb, err := database.New(dbPath)
	txChan := make(chan *bytes.Buffer, 2)
	bus.Subscribe(string(topics.Tx), txChan)

	w, err := wallet.New(rand.Read, 2, wdb, wallet.GenerateDecoys, wallet.GenerateInputs, pass)
	assert.NoError(t, err)

	_, db := lite.CreateDBConnection()
	// Ensure we have a genesis block
	genesisBlock := cfg.DecodeGenesis()
	assert.NoError(t, db.Update(func(t litedb.Transaction) error {
		return t.StoreBlock(genesisBlock)
	}))

	k, err := w.ReconstructK()
	assert.NoError(t, err)
	assert.NoError(t, maintainer.Launch(bus, db, w.ConsensusKeys().BLSPubKeyBytes, zkproof.CalculateM(k), transactor.New(w, db), 10, 10, 5))

	return bus, txChan
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
