package maintainer_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	litedb "github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/dusk-network/dusk-wallet/wallet"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/stretchr/testify/assert"
)

const pass = "password"

// Test that the maintainer will properly send new stake and bid transactions, when
// one is about to expire, or if none exist.
func TestMaintainStakesAndBids(t *testing.T) {
	bus, c, p, keys, m := setupMaintainerTest(t)
	defer os.Remove("wallet.dat")
	defer os.RemoveAll("walletDB")

	// Send round update, to start the maintainer.
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, p, nil))

	// receive first txs
	txs := receiveTxs(t, c)
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
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(2, p, bl))

	// Send another round update that is within the 'offset', to trigger sending a new pair of txs
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(6, p, bl))

	// We should get another set of two txs
	txs = receiveTxs(t, c)
	// Check correctness
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {

	bus, c, p, _, _ := setupMaintainerTest(t)
	defer os.Remove("wallet.dat")
	defer os.RemoveAll("walletDB")

	// Send round update, to start the maintainer.
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, p, nil))

	// receive first txs
	_ = receiveTxs(t, c)

	// Update round
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(2, p, nil))

	select {
	case <-c:
		t.Fatal("was not supposed to get another tx in txChan")
	case <-time.After(2 * time.Second):
		// success
	}
}

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, chan rpcbus.Request, *user.Provisioners, key.ConsensusKeys, ristretto.Scalar) {
	// Initial setup
	bus := eventbus.New()
	rpcBus := rpcbus.New()

	tr, err := transactor.New(bus, rpcBus, nil, nil, wallet.GenerateDecoys, wallet.GenerateInputs, true)
	if err != nil {
		panic(err)
	}
	go tr.Listen()

	os.Remove(cfg.Get().Wallet.File)
	assert.NoError(t, createWallet(rpcBus, pass))

	time.Sleep(100 * time.Millisecond)

	w, err := tr.Wallet()
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
	m, err := maintainer.New(bus, rpcBus, w.ConsensusKeys().BLSPubKeyBytes, mScalar, 10, 10, 5)
	assert.NoError(t, err)
	go m.Listen()

	// Mock provisioners, and insert our wallet values
	p, _ := consensus.MockProvisioners(10)
	// Note: we don't need to mock the bidlist as we should not be included if we want to trigger a bid transaction

	c := make(chan rpcbus.Request, 1)
	rpcBus.Register(rpcbus.SendMempoolTx, c)

	return bus, c, p, w.ConsensusKeys(), mScalar
}

func receiveTxs(t *testing.T, c chan rpcbus.Request) []transactions.Transaction {
	var txs []transactions.Transaction
	for i := 0; i < 2; i++ {
		r := <-c
		tx, err := marshalling.UnmarshalTx(&r.Params)
		assert.NoError(t, err)
		txs = append(txs, tx)
		r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
	}

	return txs
}

func createWallet(rpcBus *rpcbus.RPCBus, password string) error {
	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, password); err != nil {
		return err
	}

	_, err := rpcBus.Call(rpcbus.CreateWallet, rpcbus.NewRequest(*buf), 0)
	return err
}
