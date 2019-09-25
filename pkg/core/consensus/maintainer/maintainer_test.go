package maintainer_test

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	litedb "github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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
	bus, txChan, p, keys, m := setupMaintainerTest(t)
	defer os.RemoveAll(dbPath)
	defer os.Remove("wallet.dat")

	// receive first txs
	txs := receiveTxs(t, txChan)
	// Check correctness
	// We get the bid first
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)

	// add ourselves to the provisioners and the bidlist,
	// so that the maintainer gets the proper ending heights.
	// Provisioners
	member := consensus.MockMember(keys)
	member.Stakes[0].EndHeight = 10
	p.Set.Insert(member.PublicKeyBLS)
	p.Members[string(member.PublicKeyBLS)] = member
	// Bidlist
	bl := make([]user.Bid, 1)
	var mArr [32]byte
	copy(mArr[:], m.Bytes())
	bid := user.Bid{mArr, mArr, 10}
	bl[0] = bid
	// Then, send a round update to update the values on the maintainer
	bus.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(2, p, bl))

	// Send another round update that is within the 'offset', to trigger sending a new pair of txs
	bus.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(6, p, bl))

	// We should get another set of two txs
	txs = receiveTxs(t, txChan)
	// Check correctness
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {
	bus, txChan, p, _, _ := setupMaintainerTest(t)
	defer os.RemoveAll(dbPath)
	defer os.Remove("wallet.dat")

	// receive first txs
	_ = receiveTxs(t, txChan)

	// Update round
	bus.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(2, p, nil))

	select {
	case <-txChan:
		t.Fatal("was not supposed to get another tx in txChan")
	case <-time.After(2 * time.Second):
		// success
	}
}

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, chan *bytes.Buffer, *user.Provisioners, user.Keys, ristretto.Scalar) {
	// Initial setup
	bus := eventbus.New()
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
	mScalar := zkproof.CalculateM(k)
	m, err := maintainer.New(bus, w.ConsensusKeys().BLSPubKeyBytes, mScalar, transactor.New(w, db), 10, 10, 5)
	assert.NoError(t, err)
	go m.Listen()

	// Mock provisioners, and insert our wallet values
	p, _ := consensus.MockProvisioners(10)
	// Note: we don't need to mock the bidlist as we should not be included if we want to trigger a bid transaction

	// Send round update, to start the maintainer.
	bus.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(1, p, nil))

	return bus, txChan, p, w.ConsensusKeys(), mScalar
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
