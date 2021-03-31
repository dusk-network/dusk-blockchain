// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactor

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
)

var (
	log = logger.WithField("prefix", "transactor") //nolint

	errWalletNotLoaded     = errors.New("wallet is not loaded yet") //nolint
	errWalletAlreadyLoaded = errors.New("wallet is already loaded") //nolint
)

func (t *Transactor) handleAddress() (*node.LoadResponse, error) {
	// sk := t.w.SecretKey
	/*
		if sk.IsEmpty() {
			return nil, errors.New("SecretKey is not set")
		}
	*/
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

func (t *Transactor) handleSendBidTx(req *node.BidRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// create and sign transaction
	log.
		WithField("amount", req.Amount).
		WithField("locktime", req.Locktime).
		Tracef("Creating a bid tx")

	// TODO context should be created from the parent one
	ctx := context.Background()

	// FIXME: 476 - here we need to create K, EdPk; retrieve seed somehow and decide
	// an ExpirationHeight (most likely the last 2 should be retrieved from he DB)
	// Create the Ed25519 Keypair
	// XXX: We need to get the proper values, and not just make some up out of thin air.
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	pkR := &keys.StealthAddress{
		RG:  make([]byte, 32),
		PkR: make([]byte, 32),
	}

	if _, err := rand.Read(pkR.RG); err != nil {
		return nil, err
	}

	if _, err := rand.Read(pkR.PkR); err != nil {
		return nil, err
	}

	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return nil, err
	}

	tx, err := t.proxy.Provider().NewBid(ctx, k, req.Amount, secret, pkR, seed, 0, 0)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendBidTx, failed to create NewBidTx")
		return nil, err
	}

	if err = t.db.Update(func(t database.Transaction) error {
		var height uint64
		height, err = t.FetchCurrentHeight()
		if err != nil {
			return err
		}

		return t.StoreBidValues(secret, k, tx.BidTreeStorageIndex, height+250000)
	}); err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendBidTx, failed to store bid values")
		return nil, err
	}

	hash, err := t.publishTx(tx.Tx)
	if err != nil {
		// DB operations should never fail. If for some reason during runtime we
		// can no longer use the DB, we should panic.
		// TODO: maybe we can figure out a recovery procedure if this is the case.
		log.
			WithError(err).
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Panic("handleSendBidTx, failed to create publishTx")
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
	tx, err := t.proxy.Provider().NewStake(ctx, blsKey.Marshal(), req.Amount)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("locktime", req.Locktime).
			Error("handleSendStakeTx, failed to create NewStakeTx")
		return nil, err
	}

	hash, err := t.publishTx(tx)
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

	tx, err := t.proxy.Provider().NewTransfer(ctx, req.Amount, pb)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("address", string(req.Address)).
			Error("handleSendStandardTx, failed to create NewTransactionTx")
		return nil, err
	}

	// Publish transaction to the mempool processing
	hash, err := t.publishTx(tx)
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

	ub, lb, err := t.proxy.Provider().GetBalance(ctx, t.w.ViewKey)
	if err != nil {
		return nil, err
	}

	log.Tracef("wallet balance: %d, mempool balance: %d", ub, 0)
	return &node.BalanceResponse{UnlockedBalance: ub, LockedBalance: lb}, nil
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

func (t *Transactor) publishTx(tx transactions.ContractCall) ([]byte, error) {
	hash, err := tx.CalculateHash()
	if err != nil {
		return nil, err
	}

	_, err = t.rb.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 5*time.Second)
	return hash, err
}

func (t *Transactor) handleSendContract(c *node.CallContractRequest) (*node.TransactionResponse, error) {
	/*
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
	*/
	return nil, nil
}

func loadResponseFromPub(pubKey keys.PublicKey) *node.LoadResponse {
	pk := &node.PubKey{PublicKey: pubKey.ToAddr()}
	return &node.LoadResponse{Key: pk}
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
