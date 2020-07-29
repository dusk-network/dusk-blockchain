package transactor

import (
	"context"
	"errors"
	"os"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
)

var (
	log = logger.WithField("prefix", "transactor") //nolint

	errWalletNotLoaded     = errors.New("wallet is not loaded yet") //nolint
	errWalletAlreadyLoaded = errors.New("wallet is already loaded") //nolint
)

func (t *Transactor) handleCreateWallet(req *node.CreateRequest) (*node.LoadResponse, error) {
	//return err if user sends no seed
	if len(req.Seed) < 64 {
		return nil, errors.New("seed must be at least 64 bytes in size")
	}

	//create wallet with seed and pass
	err := t.createWallet(nil, req.Password)
	if err != nil {
		return nil, err
	}

	t.launchConsensus()

	return loadResponseFromPub(t.w.PublicKey), nil
}

func (t *Transactor) handleAddress() (*node.LoadResponse, error) {
	sk := t.w.SecretKey
	if sk.IsEmpty() {
		return nil, errors.New("SecretKey is not set")
	}

	return loadResponseFromPub(t.w.PublicKey), nil
}

func (t *Transactor) handleGetTxHistory() (*node.TxHistoryResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	records, err := t.w.FetchTxHistory()
	if err != nil {
		return nil, err
	}

	resp := &node.TxHistoryResponse{Records: []*node.TxRecord{}}

	for i, record := range records {
		view := record.View()
		resp.Records[i] = &node.TxRecord{
			Height:       view.Height,
			Timestamp:    view.Timestamp,
			Direction:    node.Direction(view.Direction),
			Type:         node.TxType(int32(view.Type)),
			Amount:       view.Amount,
			Fee:          view.Fee,
			UnlockHeight: view.Timelock,
			Hash:         view.Hash,
			Data:         view.Data,
			Obfuscated:   view.Obfuscated,
		}
	}

	return resp, nil
}

func (t *Transactor) handleLoadWallet(req *node.LoadRequest) (*node.LoadResponse, error) {
	if t.w != nil {
		return nil, errWalletAlreadyLoaded
	}

	pubKey, err := t.loadWallet(req.Password)
	if err != nil {
		return nil, err
	}

	t.launchConsensus()

	return loadResponseFromPub(pubKey), nil
}

func (t *Transactor) handleCreateFromSeed(req *node.CreateRequest) (*node.LoadResponse, error) {
	if t.w != nil {
		return nil, errWalletAlreadyLoaded
	}

	err := t.createWallet(req.Seed, req.Password)

	t.launchConsensus()

	return loadResponseFromPub(t.w.PublicKey), err
}

func (t *Transactor) handleSendBidTx(req *node.BidRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// // create and sign transaction
	log.
		WithField("amount", req.Amount).
		WithField("locktime", req.Locktime).
		Tracef("Creating a bid tx")

	// TODO context should be created from the parent one
	ctx := context.Background()

	txReq := transactions.MakeGenesisTxRequest(t.w.SecretKey, req.Amount, req.Fee, true)
	// FIXME: 476 - here we need to create K, EdPk; retrieve seed somehow and decide
	// an ExpirationHeight (most likely the last 2 should be retrieved from he DB)
	// Create the Ed25519 Keypair
	tx, err := t.proxy.Provider().NewBidTx(ctx, nil, nil, nil, uint64(0), txReq)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendBidTx, failed to create NewBidTx")
		return nil, err
	}

	// TODO: store the K and D in storage for the block generator

	hash, err := t.publishTx(&tx)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendBidTx, failed to create publishTx")
		return nil, err
	}

	return &node.TransactionResponse{Hash: hash}, nil
}

func (t *Transactor) handleSendStakeTx(req *node.StakeRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// create and sign transaction
	log.
		WithField("amount", req.Amount).
		WithField("locktime", req.Locktime).
		Tracef("Creating a stake tx")

	blsKey := t.w.Keys().BLSPubKey
	if blsKey == nil {
		return nil, errWalletNotLoaded
	}

	// TODO: use a parent context
	ctx := context.Background()
	// FIXME: 476 - we should calculate the expirationHeight somehow (by asking
	// the chain for the last block through the RPC bus and calculating the
	// height)
	var expirationHeight uint64
	tx, err := t.proxy.Provider().NewStakeTx(ctx, blsKey.Marshal(), expirationHeight, transactions.MakeGenesisTxRequest(t.w.SecretKey, req.Amount, req.Fee, false))
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendStakeTx, failed to create NewStakeTx")
		return nil, err
	}

	hash, err := t.publishTx(&tx)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendStakeTx, failed to create publishTx")
		return nil, err
	}

	return &node.TransactionResponse{Hash: hash}, nil
}

func (t *Transactor) handleSendStandardTx(req *node.TransferRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// create and sign transaction
	log.
		WithField("amount", req.Amount).
		WithField("address", string(req.Address)).
		Tracef("Create a standard tx")

	ctx := context.Background()

	pb, err := DecodeAddressToPublicKey(req.Address)
	if err != nil {
		return nil, err
	}

	txReq := transactions.MakeTxRequest(t.w.SecretKey, pb, req.Amount, req.Fee, false)
	tx, err := t.proxy.Provider().NewTransactionTx(ctx, txReq)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("address", string(req.Address)).
			Error("handleSendStandardTx, failed to create NewTransactionTx")
		return nil, err
	}

	// Publish transaction to the mempool processing
	hash, err := t.publishTx(&tx)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("address", string(req.Address)).
			Error("handleSendStandardTx, failed to create publishTx")
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
	balance, err := t.proxy.Provider().GetBalance(ctx, t.w.ViewKey)
	if err != nil {
		return nil, err
	}

	log.Tracef("wallet balance: %d, mempool balance: %d", balance, 0)

	return &node.BalanceResponse{UnlockedBalance: balance, LockedBalance: 0}, nil
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
	tx, err := t.proxy.Provider().NewContractCall(ctx, c.Data, txReq)
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

func loadResponseFromPub(pubKey transactions.PublicKey) *node.LoadResponse {
	pk := &node.PubKey{PublicKey: pubKey.ToAddr()}
	return &node.LoadResponse{Key: pk}
}

//nolint:unused
func (t *Transactor) launchConsensus() {
	log.Tracef("Launch consensus")
	go initiator.LaunchConsensus(context.Background(), t.eb, t.rb, t.w, t.proxy)
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
