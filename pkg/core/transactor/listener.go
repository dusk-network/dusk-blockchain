package transactor

import (
	"bytes"
	"errors"
	"fmt"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiator"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	logger "github.com/sirupsen/logrus"
)

var (
	log = logger.WithFields(logger.Fields{"prefix": "transactor"})

	errWalletNotLoaded     = errors.New("wallet is not loaded yet")
	errWalletAlreadyLoaded = errors.New("wallet is already loaded")
)

func (t *Transactor) Listen() {
	for {
		select {

		// Wallet requests to respond to
		case r := <-t.createWalletChan:
			handleRequest(r, t.handleCreateWallet, "CreateWallet")
		case r := <-t.createFromSeedChan:
			handleRequest(r, t.handleCreateFromSeed, "CreateWalletFromSeed")
		case r := <-t.loadWalletChan:
			handleRequest(r, t.handleLoadWallet, "LoadWallet")

		// Transaction requests to respond to
		case r := <-t.sendBidTxChan:
			handleRequest(r, t.handleSendBidTx, "BidTx")
		case r := <-t.sendStakeTxChan:
			handleRequest(r, t.handleSendStakeTx, "StakeTx")
		case r := <-t.sendStandardTxChan:
			handleRequest(r, t.handleSendStandardTx, "StandardTx")
		case r := <-t.getBalanceChan:
			handleRequest(r, t.handleBalance, "Balance")

		// Event list to handle
		case b := <-t.acceptedBlockChan:
			t.onAcceptedBlockEvent(b)
		}
	}
}

func handleRequest(r rpcbus.Request, handler func(r rpcbus.Request) error, name string) {

	log.Infof("Handling %s request", name)

	if err := handler(r); err != nil {
		log.Errorf("Failed %s request: %v", name, err)
		r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
		return
	}

	log.Infof("Handled %s request", name)
}

func (t *Transactor) handleCreateWallet(r rpcbus.Request) error {
	if t.w != nil {
		return errWalletAlreadyLoaded
	}

	var password string
	password, err := encoding.ReadString(&r.Params)
	if err != nil {
		return err
	}

	pubKey, err := t.createWallet(password)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, pubKey); err != nil {
		return err
	}

	t.launchConsensus()

	r.RespChan <- rpcbus.Response{*buf, nil}

	return nil
}

func (t *Transactor) handleLoadWallet(r rpcbus.Request) error {
	if t.w != nil {
		return errWalletAlreadyLoaded
	}

	var password string
	password, err := encoding.ReadString(&r.Params)
	if err != nil {
		return err
	}

	pubKey, err := t.loadWallet(password)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, pubKey); err != nil {
		return err
	}

	// Sync with genesis
	if _, err := t.w.GetSavedHeight(); err != nil {
		t.w.UpdateWalletHeight(0)
		b := cfg.DecodeGenesis()
		// call wallet.CheckBlock
		if _, _, err := t.w.CheckWireBlock(*b, true); err != nil {
			return fmt.Errorf("error checking block: %v", err)
		}
	}

	t.launchConsensus()

	r.RespChan <- rpcbus.Response{*buf, nil}

	return nil
}

func (t *Transactor) handleCreateFromSeed(r rpcbus.Request) error {
	if t.w != nil {
		return errWalletAlreadyLoaded
	}

	var seed string
	seed, err := encoding.ReadString(&r.Params)
	if err != nil {
		return err
	}

	var password string
	password, err = encoding.ReadString(&r.Params)
	if err != nil {
		return err
	}

	pubKey, err := t.createFromSeed(seed, password)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, pubKey); err != nil {
		return err
	}

	t.launchConsensus()

	r.RespChan <- rpcbus.Response{*buf, nil}

	return nil
}

func (t *Transactor) handleSendBidTx(r rpcbus.Request) error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	// read tx parameters
	var amount uint64
	if err := encoding.ReadUint64LE(&r.Params, &amount); err != nil {
		return err
	}

	var lockTime uint64
	if err := encoding.ReadUint64LE(&r.Params, &lockTime); err != nil {
		return err
	}

	// create and sign transaction
	log.Tracef("Create a bid tx (%d,%d)", amount, lockTime)

	tx, err := t.CreateBidTx(amount, lockTime)
	if err != nil {
		return err
	}

	//  Publish transaction to the mempool processing
	txid, err := t.publishTx(tx)
	if err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*bytes.NewBuffer(txid), nil}
	return nil
}

func (t *Transactor) handleSendStakeTx(r rpcbus.Request) error {

	if t.w == nil {
		return errWalletNotLoaded
	}

	// read tx parameters
	var amount uint64
	if err := encoding.ReadUint64LE(&r.Params, &amount); err != nil {
		return err
	}

	var lockTime uint64
	if err := encoding.ReadUint64LE(&r.Params, &lockTime); err != nil {
		return err
	}

	// create and sign transaction
	log.Tracef("Create a stake tx (%d,%d)", amount, lockTime)

	tx, err := t.CreateStakeTx(amount, lockTime)
	if err != nil {
		return err
	}

	//  Publish transaction to the mempool processing
	txid, err := t.publishTx(tx)
	if err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*bytes.NewBuffer(txid), nil}

	return nil
}

func (t *Transactor) handleSendStandardTx(r rpcbus.Request) error {

	if t.w == nil {
		return errWalletNotLoaded
	}

	var amount uint64
	if err := encoding.ReadUint64LE(&r.Params, &amount); err != nil {
		return err
	}

	var destPubKey string
	destPubKey, err := encoding.ReadString(&r.Params)
	if err != nil {
		return err
	}

	// create and sign transaction
	log.Tracef("Create a standard tx (%d,%s)", amount, destPubKey)

	tx, err := t.CreateStandardTx(amount, destPubKey)
	if err != nil {
		return err
	}

	//  Publish transaction to the mempool processing
	txid, err := t.publishTx(tx)
	if err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*bytes.NewBuffer(txid), nil}

	return nil
}

func (t *Transactor) handleBalance(r rpcbus.Request) error {

	if t.w == nil {
		return errWalletNotLoaded
	}

	walletBalance, mempoolBalance, err := t.Balance()
	if err != nil {
		return err
	}

	log.Tracef("wallet balance: %d, mempool balance: %d", walletBalance, mempoolBalance)

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, uint64(walletBalance)); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(buf, uint64(mempoolBalance)); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*buf, nil}
	return nil
}

func (t *Transactor) publishTx(tx transactions.Transaction) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, tx); err != nil {
		return nil, fmt.Errorf("error encoding transaction: %v\n", err)
	}

	hash, err := tx.CalculateHash()
	if err != nil {
		// If we found a valid bid tx, we should under no circumstance have issues marshalling it
		return nil, fmt.Errorf("error encoding transaction: %v\n", err)
	}

	t.eb.Publish(topics.Tx, buf)

	return hash, nil
}

func (t *Transactor) onAcceptedBlockEvent(b block.Block) {

	if t.w == nil {
		return
	}

	if err := t.syncWallet(); err != nil {
		log.Tracef("syncing failed with err: %v", err)
	}
}

func (t *Transactor) launchConsensus() {
	if !t.walletOnly {
		log.Tracef("Launch consensus")
		go initiator.LaunchConsensus(t.eb, t.rb, t.w, t.c)
	}
}
