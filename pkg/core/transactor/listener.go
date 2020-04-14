package transactor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
)

var (
	log = logger.WithFields(logger.Fields{"prefix": "transactor"})

	errWalletNotLoaded     = errors.New("wallet is not loaded yet")
	errWalletAlreadyLoaded = errors.New("wallet is already loaded")
)

func loadResponse(pubKey []byte) *node.LoadResponse {
	pk := &node.PubKey{PublicKey: pubKey}
	return &node.LoadResponse{Key: pk}
}

// Listen to Wallet requests
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
		r.RespChan <- rpcbus.Response{Resp: bytes.Buffer{}, Err: err}
		return
	}

	log.Infof("Handled %s request", name)
	r.RespChan <- rpcbus.Response{Resp: bytes.Buffer{}}
}

func (t *Transactor) handleCreateWallet(r rpcbus.Request) error {
	if t.w != nil {
		return errWalletAlreadyLoaded
	}

	req := r.Params.(*node.CreateRequest)

	pubKey, err := t.createWallet(req.Password)
	if err != nil {
		return err
	}

	t.launchConsensus()

	r.RespChan <- rpcbus.Response{Resp: loadResponse([]byte(pubKey)), Err: nil}

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

	resp := &node.LoadResponse{Key: &node.PubKey{PublicKey: []byte(addr)}}
	r.RespChan <- rpcbus.Response{Resp: resp, Err: nil}
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

	resp := &node.TxHistoryResponse{Records: make([]*node.TxRecord, len(records))}
	for i, record := range records {
		resp.Records[i] = &node.TxRecord{
			Direction:    node.Direction(record.Direction),
			Timestamp:    record.Timestamp,
			Height:       record.Height,
			Type:         node.TxType(record.TxType),
			Amount:       record.Amount,
			UnlockHeight: record.UnlockHeight,
		}
	}

	r.RespChan <- rpcbus.Response{Resp: resp, Err: nil}
	return nil
}

func (t *Transactor) handleLoadWallet(r rpcbus.Request) error {
	if t.w != nil {
		return errWalletAlreadyLoaded
	}

	req := r.Params.(*node.LoadRequest)
	pubKey, err := t.loadWallet(req.Password)
	if err != nil {
		return err
	}

	// Sync with genesis if this is a new wallet
	if _, err := t.w.GetSavedHeight(); err != nil {
		_ = t.w.UpdateWalletHeight(0)
		b := cfg.DecodeGenesis()
		// call wallet.CheckBlock
		if _, _, err := t.w.CheckWireBlock(*b); err != nil {
			return fmt.Errorf("error checking block: %v", err)
		}
	}

	t.launchConsensus()
	resp := &node.LoadResponse{Key: &node.PubKey{PublicKey: []byte(pubKey)}}
	r.RespChan <- rpcbus.Response{Resp: resp, Err: nil}

	return nil
}

func (t *Transactor) handleCreateFromSeed(r rpcbus.Request) error {
	if t.w != nil {
		return errWalletAlreadyLoaded
	}

	req := r.Params.(*node.CreateRequest)
	pubKey, err := t.createFromSeed(string(req.Seed), req.Password)
	if err != nil {
		return err
	}

	t.launchConsensus()

	r.RespChan <- rpcbus.Response{Resp: &node.LoadResponse{Key: &node.PubKey{PublicKey: []byte(pubKey)}}, Err: nil}

	return nil
}

func (t *Transactor) handleSendBidTx(r rpcbus.Request) error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	req := r.Params.(*node.ConsensusTxRequest)
	// create and sign transaction
	log.Tracef("Create a bid tx (%d,%d)", req.Amount, req.LockTime)

	tx, err := t.CreateBidTx(req.Amount, req.LockTime)
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

	r.RespChan <- rpcbus.Response{Resp: &node.TransferResponse{Hash: txid}, Err: nil}
	return nil
}

func (t *Transactor) handleSendStakeTx(r rpcbus.Request) error {

	if t.w == nil {
		return errWalletNotLoaded
	}

	req := r.Params.(*node.ConsensusTxRequest)
	// create and sign transaction
	log.Tracef("Create a stake tx (%d,%d)", req.Amount, req.LockTime)

	tx, err := t.CreateStakeTx(req.Amount, req.LockTime)
	if err != nil {
		return err
	}

	//  Publish transaction to the mempool processing
	txid, err := t.publishTx(tx)
	if err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{Resp: &node.TransferResponse{Hash: txid}, Err: nil}

	return nil
}

func (t *Transactor) handleSendStandardTx(r rpcbus.Request) error {

	if t.w == nil {
		return errWalletNotLoaded
	}

	req := r.Params.(*node.TransferRequest)
	// create and sign transaction
	log.Tracef("Create a standard tx (%d,%s)", req.Amount, string(req.Address))

	tx, err := t.CreateStandardTx(req.Amount, string(req.Address))
	if err != nil {
		return err
	}

	//  Publish transaction to the mempool processing
	txid, err := t.publishTx(tx)
	if err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{Resp: &node.TransferResponse{Hash: txid}, Err: nil}

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

	r.RespChan <- rpcbus.Response{Resp: &node.BalanceResponse{UnlockedBalance: unlockedBalance, LockedBalance: lockedBalance}, Err: nil}
	return nil
}

func (t *Transactor) handleUnconfirmedBalance(r rpcbus.Request) error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	// Retrieve mempool txs
	resp, err := t.rb.Call(topics.GetMempoolTxs, rpcbus.Request{Params: bytes.Buffer{}, RespChan: make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		return err
	}
	txs := resp.([]transactions.Transaction)

	unconfirmedBalance, err := t.w.CheckUnconfirmedBalance(txs)
	if err != nil {
		return err
	}

	log.Tracef("unconfirmed wallet balance: %d", unconfirmedBalance)

	r.RespChan <- rpcbus.Response{Resp: &node.BalanceResponse{UnlockedBalance: unconfirmedBalance}, Err: nil}
	return nil
}

func (t *Transactor) handleAutomateConsensusTxs(r rpcbus.Request) error {
	if err := t.launchMaintainer(); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{Resp: &node.GenericResponse{Response: "Consensus transactions are now being automated."}, Err: nil}
	return nil
}

func (t *Transactor) handleClearWalletDatabase(r rpcbus.Request) error {
	if t.w == nil {
		if err := os.RemoveAll(cfg.Get().Wallet.Store); err != nil {
			return err
		}

		r.RespChan <- rpcbus.Response{Resp: &node.GenericResponse{Response: "Wallet database deleted."}, Err: nil}
		return nil
	}

	if err := t.w.ClearDatabase(); err != nil {
		return err
	}

	r.RespChan <- rpcbus.Response{Resp: &node.GenericResponse{Response: "Wallet database deleted."}, Err: nil}
	return nil
}

func (t *Transactor) handleIsWalletLoaded(r rpcbus.Request) error {
	r.RespChan <- rpcbus.Response{Resp: &node.WalletStatusResponse{Loaded: t.w != nil}, Err: nil}
	return nil
}

func (t *Transactor) publishTx(tx transactions.Transaction) ([]byte, error) {
	hash, err := tx.CalculateHash()
	if err != nil {
		// If we found a valid bid tx, we should under no circumstance have issues marshaling it
		return nil, fmt.Errorf("error encoding transaction: %v", err)
	}

	_, err = t.rb.Call(topics.SendMempoolTx, rpcbus.Request{Params: tx, RespChan: make(chan rpcbus.Response, 1)}, 0)
	if err != nil {
		return nil, err
	}

	return hash, nil
}

//nolint:unparam
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

		return tr.StoreBidValues(tx.StandardTx().Outputs[0].Commitment.Bytes(), k.Bytes(), tx.LockTime())
	})
}
