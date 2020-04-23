package transactor

import (
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
)

var (
	log = logger.WithFields(logger.Fields{"prefix": "transactor"}) //nolint

	errWalletNotLoaded     = errors.New("wallet is not loaded yet") //nolint
	errWalletAlreadyLoaded = errors.New("wallet is already loaded") //nolint
)

//nolint
func loadResponse(pubKey []byte) *node.LoadResponse {
	pk := &node.PubKey{PublicKey: pubKey}
	return &node.LoadResponse{Key: pk}
}

func (t *Transactor) handleCreateWallet(req *node.CreateRequest) (*node.LoadResponse, error) {
	// TODO: RUSK call
	// pubKey, err := t.createWallet(req.Password)
	// if err != nil {
	// 	return nil, err
	// }

	// // TODO: will this still make sense after the migration
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil
	return nil, nil
}

func (t *Transactor) handleAddress() (*node.LoadResponse, error) {
	// TODO: RUSK call
	// addr, err := t.w.PublicAddress()
	// if err != nil {
	// 	return nil, err
	// }

	// return loadResponse([]byte(addr)), nil
	return nil, nil
}

func (t *Transactor) handleGetTxHistory() (*node.TxHistoryResponse, error) {
	// TODO: RUSK call
	// records, err := t.w.FetchTxHistory()
	// if err != nil {
	// 	return err
	// }

	// resp := &node.TxHistoryResponse{Records: make([]*node.TxRecord, len(records))}
	// for i, record := range records {
	// 	resp.Records[i] = &node.TxRecord{
	// 		Direction:    node.Direction(record.Direction),
	// 		Timestamp:    record.Timestamp,
	// 		Height:       record.Height,
	// 		Type:         node.TxType(record.TxType),
	// 		Amount:       record.Amount,
	// 		UnlockHeight: record.UnlockHeight,
	// 	}
	// }

	// return resp, nil
	return nil, nil
}

func (t *Transactor) handleLoadWallet(req *node.LoadRequest) (*node.LoadResponse, error) {
	// if t.w != nil {
	// 	return nil, errWalletAlreadyLoaded
	// }

	// pubKey, err := t.loadWallet(req.Password)
	// if err != nil {
	// 	return nil, err
	// }

	// // TODO: will this still make sense after migration?
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil
	return nil, nil
}

func (t *Transactor) handleCreateFromSeed(req *node.CreateRequest) (*node.LoadResponse, error) {
	// TODO: RUSK call
	// if t.w != nil {
	// 	return nil, errWalletAlreadyLoaded
	// }

	// pubKey, err := t.createFromSeed(string(req.Seed), req.Password)
	// if err != nil {
	// 	return nil, err
	// }

	// // TODO: will this still make sense after migration?
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil
	return nil, nil
}

func (t *Transactor) handleSendBidTx(req *node.BidRequest) (*node.TransactionResponse, error) {
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// // create and sign transaction
	// log.Tracef("Create a bid tx (%d,%d)", req.Amount, req.LockTime)

	// // TODO: RUSK call
	// tx, err := t.CreateBidTx(req.Amount, req.LockTime)
	// if err != nil {
	// 	return nil, err
	// }

	// //  Publish transaction to the mempool processing
	// txid, err := t.publishTx(tx)
	// if err != nil {
	// 	return nil, err
	// }

	// // Save relevant values in the database for the generation component to use
	// if err := t.writeBidValues(tx); err != nil {
	// 	return nil, err
	// }

	// return &node.TransferResponse{Hash: txid}, nil
	return nil, nil
}

func (t *Transactor) handleSendStakeTx(req *node.StakeRequest) (*node.TransactionResponse, error) {
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// // create and sign transaction
	// log.Tracef("Create a stake tx (%d,%d)", req.Amount, req.LockTime)

	// // TODO: RUSK call
	// tx, err := t.CreateStakeTx(req.Amount, req.LockTime)
	// if err != nil {
	// 	return nil, err
	// }

	// //  Publish transaction to the mempool processing
	// txid, err := t.publishTx(tx)
	// if err != nil {
	// 	return nil, err
	// }

	// return &node.TransferResponse{Hash: txid}, nil
	return nil, nil
}

func (t *Transactor) handleSendStandardTx(req *node.TransferRequest) (*node.TransactionResponse, error) {
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// // create and sign transaction
	// log.Tracef("Create a standard tx (%d,%s)", req.Amount, string(req.Address))

	// // TODO: RUSK call
	// tx, err := t.CreateStandardTx(req.Amount, string(req.Address))
	// if err != nil {
	// 	return nil, err
	// }

	// //  Publish transaction to the mempool processing
	// txid, err := t.publishTx(tx)
	// if err != nil {
	// 	return nil, err
	// }

	// return &node.TransferResponse{Hash: txid}, nil
	return nil, nil
}

func (t *Transactor) handleBalance() (*node.BalanceResponse, error) {
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// // TODO: RUSK call
	// unlockedBalance, lockedBalance, err := t.Balance()
	// if err != nil {
	// 	return nil, err
	// }

	// log.Tracef("wallet balance: %d, mempool balance: %d", unlockedBalance, lockedBalance)

	// return &node.BalanceResponse{UnlockedBalance: unlockedBalance, LockedBalance: lockedBalance}, nil
	return nil, nil
}

func (t *Transactor) handleClearWalletDatabase() (*node.GenericResponse, error) {
	// if t.w == nil {
	// 	if err := os.RemoveAll(cfg.Get().Wallet.Store); err != nil {
	// 		return nil, err
	// 	}

	// 	return node.GenericResponse{Response: "Wallet database deleted."}, nil
	// }

	// if err := t.w.ClearDatabase(); err != nil {
	// 	return nil, err
	// }

	// return &node.GenericResponse{Response: "Wallet database deleted."}, nil
	return nil, nil
}

func (t *Transactor) handleIsWalletLoaded() (*node.WalletStatusResponse, error) {
	// return &node.WalletStatusResponse{Loaded: t.w != nil}, nil
	return nil, nil
}

//nolint:unused
func (t *Transactor) publishTx(tx transactions.Transaction) ([]byte, error) {
	// hash, err := tx.CalculateHash()
	// if err != nil {
	// 	// If we found a valid bid tx, we should under no circumstance have issues marshaling it
	// 	return nil, fmt.Errorf("error encoding transaction: %v", err)
	// }

	// // TODO: this can go over the event bus
	// _, err = t.rb.Call(topics.SendMempoolTx, rpcbus.Request{Params: tx, RespChan: make(chan rpcbus.Response, 1)}, 0)
	// if err != nil {
	// 	return nil, err
	// }

	// return hash, nil
	return nil, nil
}

//nolint:unused
func (t *Transactor) launchConsensus() {
	// if !t.walletOnly {
	// 	log.Tracef("Launch consensus")
	// 	go initiator.LaunchConsensus(t.eb, t.rb, t.w, t.c)
	// }
}

//nolint:unused
func (t *Transactor) writeBidValues(tx transactions.Transaction) error {
	// return t.db.Update(func(tr database.Transaction) error {
	// 	k, err := t.w.ReconstructK()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	return tr.StoreBidValues(tx.StandardTx().Outputs[0].Commitment.Bytes(), k.Bytes(), tx.LockTime())
	// })
	return nil
}
