package transactor

import (
	"context"
	"errors"
	"os"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

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
	// TODO: use parent context
	ctx := context.Background()
	sk, err := t.keyMaster.GenerateSecretKey(ctx, req.Seed)
	if err != nil {
		return nil, err
	}

	//set it for further use
	t.secretKey = sk

	//create wallet with seed and pass
	pubKey, _, err := t.createFromSeed(req.Seed, req.Password)
	if err != nil {
		return nil, err
	}

	// TODO: will this still make sense after the migration
	// t.launchConsensus()

	return &node.LoadResponse{Key: &node.PubKey{
		PublicKey: pubKey.ToAddr(),
	}}, nil
}

func (t *Transactor) handleAddress() (*node.LoadResponse, error) {
	sk := t.w.SecretKey
	if sk.IsEmpty() {
		return nil, errors.New("SecretKey is not set")
	}

	return &node.LoadResponse{Key: &node.PubKey{
		PublicKey: t.w.PublicKey.ToAddr(),
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

	pubKey, _, err := t.loadWallet(req.Password)
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
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// // create and sign transaction
	log.Tracef("Create a bid tx (%d,%d)", req.Amount, req.Locktime)

	// TODO context should be created from the parent one
	ctx := context.Background()

	// TODO the Bid transaction needs more input
	txReq := transactions.MakeGenesisTxRequest(t.w.SecretKey, uint64(0), req.Fee, true)
	tx, err := t.provider.NewBidTx(ctx, txReq)
	if err != nil {
		return nil, err
	}

	// TODO: store the K and D in storage for the block generator

	hash, err := t.publishTx(&tx)
	if err != nil {
		return nil, err
	}

	return &node.TransactionResponse{Hash: hash}, nil
}

func (t *Transactor) handleSendStakeTx(req *node.StakeRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// create and sign transaction
	log.Tracef("Create a stake tx (%d,%d)", req.Amount, req.Locktime)

	blsKey := t.w.Keys().BLSPubKey
	if blsKey == nil {
		return nil, errWalletNotLoaded
	}

	// TODO: use a parent context
	ctx := context.Background()
	tx, err := t.provider.NewStakeTx(ctx, blsKey.Marshal(), transactions.MakeGenesisTxRequest(t.w.SecretKey, req.Amount, req.Fee, false))
	if err != nil {
		return nil, err
	}

	hash, err := t.publishTx(&tx)
	if err != nil {
		return nil, err
	}

	return &node.TransactionResponse{Hash: hash}, nil
}

func (t *Transactor) handleSendStandardTx(req *node.TransferRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// create and sign transaction
	log.Tracef("Create a standard tx (%d,%s)", req.Amount, string(req.Address))

	ctx := context.Background()

	pb, err := DecodeAddressToPublicKey(req.Address)
	if err != nil {
		return nil, err
	}

	txReq := transactions.MakeTxRequest(t.w.SecretKey, pb, req.Amount, req.Fee, false)
	tx, err := t.provider.NewTransactionTx(ctx, txReq)
	if err != nil {
		return nil, err
	}

	// Publish transaction to the mempool processing
	hash, err := t.publishTx(&tx)
	if err != nil {
		return nil, err
	}

	return &node.TransactionResponse{Hash: hash}, nil
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
	if t.w != nil && !t.w.SecretKey.IsEmpty() {
		isLoaded = true
	}
	return &node.WalletStatusResponse{Loaded: isLoaded}, nil
}

func (t *Transactor) publishTx(tx transactions.ContractCall) ([]byte, error) {
	hash, err := tx.CalculateHash()
	if err != nil {
		return nil, err
	}

	msg := message.New(topics.Tx, tx)
	t.eb.Publish(topics.Tx, msg)

	return hash, nil
}

func (t *Transactor) handleSendContract(c *node.CallContractRequest) (*node.TransactionResponse, error) {
	ctx := context.Background()

	pb, err := DecodeAddressToPublicKey(c.Address)
	if err != nil {
		return nil, err
	}

	txReq := transactions.MakeTxRequest(t.w.SecretKey, pb, uint64(0), c.Fee, true)
	tx, err := t.provider.NewContractCall(ctx, c.Data, txReq)
	if err != nil {
		return nil, err
	}

	// Publish transaction to the mempool processing
	hash, err := t.publishTx(tx)
	if err != nil {
		return nil, err
	}

	return &node.TransactionResponse{Hash: hash}, nil
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
