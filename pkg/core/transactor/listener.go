package transactor

import (
	"context"
	"errors"

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
	ctx := context.Background()
	records, err := t.walletClient.CreateWallet(ctx, &node.CreateRequest{Password: req.Password})
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after the migration
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil
	return records, nil
}

func (t *Transactor) handleAddress() (*node.LoadResponse, error) {
	ctx := context.Background()
	records, err := t.walletClient.GetAddress(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (t *Transactor) handleGetTxHistory() (*node.TxHistoryResponse, error) {
	ctx := context.Background()
	records, err := t.walletClient.GetTxHistory(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (t *Transactor) handleLoadWallet(req *node.LoadRequest) (*node.LoadResponse, error) {
	// TODO: will this still make sense after migration?
	// if t.w != nil {
	// 	return nil, errWalletAlreadyLoaded
	// }

	ctx := context.Background()
	records, err := t.walletClient.LoadWallet(ctx, &node.LoadRequest{Password: req.Password})
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after migration?
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil

	return records, nil
}

func (t *Transactor) handleCreateFromSeed(req *node.CreateRequest) (*node.LoadResponse, error) {
	// TODO: will this still make sense after migration?
	// if t.w != nil {
	// 	return nil, errWalletAlreadyLoaded
	// }

	ctx := context.Background()
	records, err := t.walletClient.CreateFromSeed(ctx, &node.CreateRequest{Password: req.Password, Seed: req.Seed})
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after migration?
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil
	return records, nil
}

func (t *Transactor) handleSendBidTx(req *node.BidRequest) (*node.TransactionResponse, error) {
	// TODO: will this still make sense after migration?
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// // create and sign transaction
	log.Tracef("Create a bid tx (%d,%d)", req.Amount, req.Locktime)

	ctx := context.Background()
	tx, err := t.transactorClient.Bid(ctx, &node.BidRequest{Amount: req.Amount, Locktime: req.Locktime, Fee: req.Fee})
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after migration?
	////  Publish transaction to the mempool processing
	//txid, err := t.publishTx(tx)
	//if err != nil {
	//	return nil, err
	//}

	// Save relevant values in the database for the generation component to use
	if err := t.writeBidValues(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (t *Transactor) handleSendStakeTx(req *node.StakeRequest) (*node.TransactionResponse, error) {
	// TODO: will this still make sense after migration?
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// create and sign transaction
	log.Tracef("Create a stake tx (%d,%d)", req.Amount, req.Locktime)

	ctx := context.Background()
	tx, err := t.transactorClient.Stake(ctx, &node.StakeRequest{Amount: req.Amount, Locktime: req.Locktime, Fee: req.Fee})
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after migration?
	// //  Publish transaction to the mempool processing
	// txid, err := t.publishTx(tx)
	// if err != nil {
	// 	return nil, err
	// }

	return tx, nil
}

func (t *Transactor) handleSendStandardTx(req *node.TransferRequest) (*node.TransactionResponse, error) {
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	// create and sign transaction
	log.Tracef("Create a standard tx (%d,%s)", req.Amount, string(req.Address))

	ctx := context.Background()
	tx, err := t.transactorClient.Transfer(ctx, &node.TransferRequest{Amount: req.Amount, Address: req.Address, Fee: req.Fee})
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after migration?
	// Publish transaction to the mempool processing
	// txid, err := t.publishTx(tx)
	// if err != nil {
	// 	return nil, err
	// }

	return tx, nil
}

func (t *Transactor) handleBalance() (*node.BalanceResponse, error) {
	// TODO: will this still make sense after migration?
	// if t.w == nil {
	// 	return nil, errWalletNotLoaded
	// }

	ctx := context.Background()
	records, err := t.walletClient.GetBalance(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	log.Tracef("wallet balance: %d, mempool balance: %d", records.UnlockedBalance, records.LockedBalance)

	return records, nil
}

func (t *Transactor) handleClearWalletDatabase() (*node.GenericResponse, error) {
	// TODO: will this still make sense after migration?
	// if t.w == nil {
	// 	if err := os.RemoveAll(cfg.Get().Wallet.Store); err != nil {
	// 		return nil, err
	// 	}

	ctx := context.Background()
	records, err := t.walletClient.ClearWalletDatabase(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (t *Transactor) handleIsWalletLoaded() (*node.WalletStatusResponse, error) {
	ctx := context.Background()

	records, err := t.walletClient.GetWalletStatus(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return records, nil
}

//nolint:unused
func (t *Transactor) publishTx(tx *node.TransactionResponse) ([]byte, error) {
	// hash, err := tx.CalculateHash()
	// if err != nil {
	// 	// If we found a valid bid tx, we should under no circumstance have issues marshaling it
	// 	return nil, fmt.Errorf("error encoding transaction: %v", err)
	// }

	// TODO: this can go over the event bus
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
func (t *Transactor) writeBidValues(tx *node.TransactionResponse) error {
	// return t.db.Update(func(tr database.Transaction) error {
	// 	k, err := t.w.ReconstructK()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	return tr.StoreBidValues(tx.StandardTx().Outputs[0].Commitment.Bytes(), k.Bytes(), tx.LockTime())
	// })
	return nil
}
