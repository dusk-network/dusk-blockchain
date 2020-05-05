package transactor

import (
	"context"
	"errors"
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"

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
	//return err if user sends no seed
	if len(req.Seed) < 64 {
		return nil, errors.New("seed must be at least 64 bytes in size")
	}

	//generate secret key with rusk
	ctx := context.Background()
	record, err := t.ruskClient.GenerateSecretKey(ctx, &rusk.GenerateSecretKeyRequest{B: req.Seed})
	if err != nil {
		return nil, err
	}

	//set it for further use
	t.secretKey = new(transactions.SecretKey)
	transactions.USecretKey(record, t.secretKey)

	//create wallet with seed and pass
	pubKey, err := t.createFromSeed(req.Seed, req.Password)
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after the migration
	// t.launchConsensus()

	// TODO: KEYS is hex encoding the PK correct?
	return &node.LoadResponse{Key: &node.PubKey{
		PublicKey: pubKey.ToAddr(),
	}}, nil
}

func (t *Transactor) handleAddress() (*node.LoadResponse, error) {
	if t.w.SecretKey() == nil {
		return nil, errors.New("SecretKey is not set")
	}

	ruskSK := new(rusk.SecretKey)
	transactions.MSecretKey(ruskSK, t.w.SecretKey())

	//get the pub key and return
	ctx := context.Background()
	pubKey, err := t.ruskClient.Keys(ctx, ruskSK)
	if err != nil {
		return nil, err
	}

	return &node.LoadResponse{Key: &node.PubKey{
		PublicKey: []byte(pubKey.String()),
	}}, nil
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

	pubKey, err := t.loadWallet(req.Password)
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after migration?
	// t.launchConsensus()
	// return loadResponse([]byte(pubKey)), nil

	return &node.LoadResponse{Key: &node.PubKey{
		PublicKey: pubKey.ToAddr(),
	}}, nil
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
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// NOTE: maybe we will separate the locked and unlocked balances
	// This call should be updated in that case
	ctx := context.Background()
	balanceResponse, err := t.walletClient.GetBalance(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	log.Tracef("wallet balance: %d, mempool balance: %d", balanceResponse.UnlockedBalance, balanceResponse.LockedBalance)

	return &node.BalanceResponse{UnlockedBalance: balanceResponse.UnlockedBalance, LockedBalance: balanceResponse.LockedBalance}, nil
}

func (t *Transactor) handleClearWalletDatabase() (*node.GenericResponse, error) {
	if t.w == nil {
		if err := os.RemoveAll(cfg.Get().Wallet.Store); err != nil {
			return nil, err
		}
	}

	if err := t.w.ClearDatabase(); err != nil {
		return nil, err
	}
	return &node.GenericResponse{Response: "Wallet database deleted."}, nil
}

func (t *Transactor) handleIsWalletLoaded() (*node.WalletStatusResponse, error) {
	isLoaded := false
	if t.w != nil && t.w.SecretKey() != nil {
		isLoaded = true
	}
	return &node.WalletStatusResponse{Loaded: isLoaded}, nil
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
