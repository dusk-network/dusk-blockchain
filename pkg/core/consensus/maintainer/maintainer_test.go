package maintainer_test

/*
const pass = "password"

// Test that the maintainer will properly send new stake and bid transactions, when
// one is about to expire, or if none exist.
func TestMaintainStakesAndBids(t *testing.T) {
	bus, c, p, keys, m := setupMaintainerTest(t)
	defer func() {
		_ = os.Remove("wallet.dat")
		_ = os.RemoveAll("walletDB")
	}()

	// Send round update, to start the maintainer.
	ru := consensus.MockRoundUpdate(1, p, nil)
	ruMsg := message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// receive first txs
	txs := receiveTxs(c)
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
	bid := user.Bid{X: mArr, M: mArr, EndHeight: 10}
	bl[0] = bid
	// Then, send a round update to update the values on the maintainer
	ru = consensus.MockRoundUpdate(2, p, bl)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// Send another round update that is within the 'offset', to trigger sending a new pair of txs
	ru = consensus.MockRoundUpdate(950, p, bl)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// We should get another set of two txs
	txs = receiveTxs(c)
	// Check correctness
	assert.True(t, txs[0].Type() == transactions.BidType)
	assert.True(t, txs[1].Type() == transactions.StakeType)
}

// Ensure the maintainer does not keep sending bids and stakes until they are included.
func TestSendOnce(t *testing.T) {
	bus, c, p, _, _ := setupMaintainerTest(t)
	defer func() {
		_ = os.Remove("wallet.dat")
		_ = os.RemoveAll("walletDB")
	}()

	// Send round update, to start the maintainer.
	ru := consensus.MockRoundUpdate(1, p, nil)
	ruMsg := message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	// receive first txs
	_ = receiveTxs(c)

	// Update round
	ru = consensus.MockRoundUpdate(2, p, nil)
	ruMsg = message.New(topics.RoundUpdate, ru)
	bus.Publish(topics.RoundUpdate, ruMsg)

	select {
	case <-c:
		t.Fatal("was not supposed to get another tx in txChan")
	case <-time.After(2 * time.Second):
		// success
	}
}

func setupMaintainerTest(t *testing.T) (*eventbus.EventBus, chan rpcbus.Request, *user.Provisioners, key.Keys, ristretto.Scalar) {
	// Initial setup
	bus := eventbus.New()
	rpcBus := rpcbus.New()

	_, db := lite.CreateDBConnection()
	// Ensure we have a genesis block
	genesisBlock := cfg.DecodeGenesis()
	assert.NoError(t, db.Update(func(t litedb.Transaction) error {
		return t.StoreBlock(genesisBlock)
	}))

	tr := transactor.New(bus, db, true, nil)
	go tr.Listen()

	_ = os.Remove(cfg.Get().Wallet.File)
	assert.NoError(t, createWallet(rpcBus, pass))

	time.Sleep(100 * time.Millisecond)

	w, err := tr.Wallet()
	assert.NoError(t, err)

	k, err := w.ReconstructK()
	assert.NoError(t, err)
	mScalar := zkproof.CalculateM(k)
	m := maintainer.New(bus, rpcBus, w.Keys().BLSPubKeyBytes, mScalar)
	go m.Listen()

	// Mock provisioners, and insert our wallet values
	p, _ := consensus.MockProvisioners(10)
	// Note: we don't need to mock the bidlist as we should not be included if we want to trigger a bid transaction

	c := make(chan rpcbus.Request, 1)
	err = rpcBus.Register(topics.SendMempoolTx, c)
	require.Nil(t, err)

	return bus, c, p, w.Keys(), mScalar
}

//nolint:unused
func receiveTxs(c chan rpcbus.Request) []transactions.Transaction {
	var txs []transactions.Transaction
	for i := 0; i < 2; i++ {
		r := <-c
		tx := r.Params.(transactions.Transaction)
		txs = append(txs, tx)
		r.RespChan <- rpcbus.NewResponse(nil, nil)
	}

	return txs
}

//nolint:unused
func createWallet(rpcBus *rpcbus.RPCBus, password string) error {
	_, err := rpcBus.Call(topics.CreateWallet, rpcbus.NewRequest(&node.CreateRequest{Password: password}), 0)
	return err
}
*/
