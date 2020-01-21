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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/stretchr/testify/assert"
)

// Test that the maintainer will properly send new stake and bid transactions, when
// one is about to expire, or if none exist.
func TestMaintainStakesAndBids(t *testing.T) {
	bus, c, p, keys, m, exitChan := setupMaintainerTest(t)
	defer os.Remove("wallet.dat")
	defer os.RemoveAll("walletDB")

	// Send round update, to start the maintainer.
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, p, nil))

	// receive first txs
	receiveTxs(t, c)

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
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(950, p, bl))

	// We should get another set of two txs
	receiveTxs(t, c)
	exitChan <- struct{}{}
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {
	bus, c, p, _, _, exitChan := setupMaintainerTest(t)
	defer os.Remove("wallet.dat")
	defer os.RemoveAll("walletDB")

	// Send round update, to start the maintainer.
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, p, nil))

	// receive first txs
	receiveTxs(t, c)

	// Update round
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(2, p, nil))

	select {
	case <-c:
		t.Fatal("was not supposed to get another tx in txChan")
	case <-time.After(1 * time.Second):
		// success
	}

	exitChan <- struct{}{}
}

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, chan struct{}, *user.Provisioners, key.ConsensusKeys, ristretto.Scalar, chan struct{}) {
	// Initial setup
	bus := eventbus.New()
	rpcBus := rpcbus.New()

	c := make(chan struct{}, 2)
	exitChan := startFakeTransactor(rpcBus, c)

	_, db := lite.CreateDBConnection()
	// Ensure we have a genesis block
	genesisBlock := cfg.DecodeGenesis()
	assert.NoError(t, db.Update(func(t litedb.Transaction) error {
		return t.StoreBlock(genesisBlock)
	}))

	var mScalar ristretto.Scalar
	mScalar.Rand()
	keys, err := key.NewRandConsensusKeys()
	if err != nil {
		panic(err)
	}

	m := maintainer.New(bus, rpcBus, keys.BLSPubKeyBytes, mScalar)
	go m.Listen()

	// Mock provisioners, and insert our wallet values
	p, _ := consensus.MockProvisioners(10)
	// Note: we don't need to mock the bidlist as we should not be included if we want to trigger a bid transaction

	return bus, c, p, keys, mScalar, exitChan
}

func startFakeTransactor(rb *rpcbus.RPCBus, txChan chan struct{}) chan struct{} {
	exitChan := make(chan struct{}, 1)
	c := make(chan rpcbus.Request, 10)
	rb.Register(rpcbus.SendBidTx, c)
	rb.Register(rpcbus.SendStakeTx, c)

	go func(c chan rpcbus.Request, exitChan chan struct{}) {
		for {
			select {
			case r := <-c:
				txChan <- struct{}{}
				r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
			case <-exitChan:
				return
			}
		}
	}(c, exitChan)

	return exitChan
}

func receiveTxs(t *testing.T, c chan struct{}) {
	for i := 0; i < 2; i++ {
		<-c
	}
}
