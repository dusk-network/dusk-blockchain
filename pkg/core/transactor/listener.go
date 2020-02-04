package transactor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/dusk-network/dusk-wallet/v2/txrecords"
	"github.com/dusk-network/dusk-wallet/v2/wallet"
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
		case r := <-t.automateConsensusTxsChan:
			handleRequest(r, t.handleAutomateConsensusTxs, "AutomateConsensusTxs")
		case r := <-t.clearWalletDatabaseChan:
			handleRequest(r, t.handleClearWalletDatabase, "ClearWalletDatabase")

		// Transaction requests to respond to
		case r := <-t.sendBidTxChan:
			handleRequest(r, t.handleSendBidTx, "BidTx")
		case r := <-t.sendStakeTxChan:
			handleRequest(r, t.handleSendStakeTx, "StakeTx")
		case r := <-t.sendStandardTxChan:
			handleRequest(r, t.handleSendStandardTx, "StandardTx")

		// Information requests to respond to
		case r := <-t.getBalanceChan:
			handleRequest(r, t.handleBalance, "Balance")
		case r := <-t.getUnconfirmedBalanceChan:
			handleRequest(r, t.handleUnconfirmedBalance, "UnconfirmedBalance")
		case r := <-t.getAddressChan:
			handleRequest(r, t.handleAddress, "Address")
		case r := <-t.getTxHistoryChan:
			handleRequest(r, t.handleGetTxHistory, "GetTxHistory")
		case r := <-t.isWalletLoadedChan:
			handleRequest(r, t.handleIsWalletLoaded, "IsWalletLoaded")

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
	r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
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

func (t *Transactor) handleAddress(r rpcbus.Request) error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	addr, err := t.w.PublicAddress()
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if _, err := buf.WriteString(addr); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*buf, nil}
	return nil
}

func (t *Transactor) handleGetTxHistory(r rpcbus.Request) error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	records, err := t.w.FetchTxHistory()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		r.RespChan <- rpcbus.Response{*bytes.NewBufferString("No records found."), nil}
		return nil
	}

	s := &strings.Builder{}
	for _, record := range records {
		// Direction
		if record.Direction == txrecords.In {
			s.WriteString("IN / ")
		} else {
			s.WriteString("OUT / ")
		}
		// Height
		s.WriteString(strconv.FormatUint(record.Height, 10) + " / ")
		// Time
		s.WriteString(time.Unix(record.Timestamp, 0).Format(time.UnixDate) + " / ")
		// Amount
		s.WriteString(fmt.Sprintf("%.8f DUSK", float64(record.Amount)/float64(wallet.DUSK)) + " / ")
		// Unlock height
		s.WriteString("Unlocks at " + strconv.FormatUint(record.UnlockHeight, 10) + " / ")
		// Recipient
		s.WriteString("Recipient: " + record.Recipient)

		s.WriteString("\n")
	}

	r.RespChan <- rpcbus.Response{*bytes.NewBufferString(s.String()), nil}
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
		if _, _, err := t.w.CheckWireBlock(*b); err != nil {
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

	// Save relevant values in the database for the generation component to use
	if err := t.writeBidValues(tx); err != nil {
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

	unlockedBalance, lockedBalance, err := t.Balance()
	if err != nil {
		return err
	}

	log.Tracef("wallet balance: %d, mempool balance: %d", unlockedBalance, lockedBalance)

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, uint64(unlockedBalance)); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(buf, uint64(lockedBalance)); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*buf, nil}
	return nil
}

func (t *Transactor) handleUnconfirmedBalance(r rpcbus.Request) error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	// Retrieve mempool txs
	txsBuf, err := t.rb.Call(rpcbus.GetMempoolTxs, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		return err
	}

	lTxs, err := encoding.ReadVarInt(&txsBuf)
	if err != nil {
		return err
	}

	txs := make([]transactions.Transaction, lTxs)
	for i := range txs {
		tx, err := message.UnmarshalTx(&txsBuf)
		if err != nil {
			return err
		}

		txs[i] = tx
	}

	unconfirmedBalance, err := t.w.CheckUnconfirmedBalance(txs)
	if err != nil {
		return err
	}

	log.Tracef("unconfirmed wallet balance: %d", unconfirmedBalance)

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, uint64(unconfirmedBalance)); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{*buf, nil}
	return nil
}

func (t *Transactor) handleAutomateConsensusTxs(r rpcbus.Request) error {
	if err := t.launchMaintainer(); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
	return nil
}

func (t *Transactor) handleClearWalletDatabase(r rpcbus.Request) error {
	if t.w == nil {
		if err := os.RemoveAll(cfg.Get().Wallet.Store); err != nil {
			return err
		}

		r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
		return nil
	}

	if err := t.w.ClearDatabase(); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
	return nil
}

func (t *Transactor) handleIsWalletLoaded(r rpcbus.Request) error {
	buf := new(bytes.Buffer)
	if err := encoding.WriteBool(buf, t.w != nil); err != nil {
		return err
	}
	r.RespChan <- rpcbus.Response{*buf, nil}
	return nil
}

func (t *Transactor) publishTx(tx transactions.Transaction) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := message.MarshalTx(buf, tx); err != nil {
		return nil, fmt.Errorf("error encoding transaction: %v\n", err)
	}

	hash, err := tx.CalculateHash()
	if err != nil {
		// If we found a valid bid tx, we should under no circumstance have issues marshalling it
		return nil, fmt.Errorf("error encoding transaction: %v\n", err)
	}

	_, err = t.rb.Call(rpcbus.SendMempoolTx, rpcbus.Request{*buf, make(chan rpcbus.Response, 1)}, 0)
	if err != nil {
		return nil, err
	}

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

func (t *Transactor) writeBidValues(tx transactions.Transaction) error {
	return t.db.Update(func(tr database.Transaction) error {
		k, err := t.w.ReconstructK()
		if err != nil {
			return err
		}

		return tr.StoreBidValues(tx.StandardTx().Outputs[0].Commitment.Bytes(), k.Bytes())
	})
}
