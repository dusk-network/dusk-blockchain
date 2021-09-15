// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactor

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
)

var (
	log = logger.WithField("process", "transactor") //nolint

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

func (t *Transactor) handleSendStakeTx(req *node.StakeRequest) (*node.TransactionResponse, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	// create and sign transaction
	log.
		WithField("amount", req.Amount).
		WithField("locktime", req.Locktime).
		Trace("Creating a stake tx")

	blsKey := t.w.Keys().BLSPubKey
	if blsKey == nil {
		return nil, errWalletNotLoaded
	}

	// TODO: use a parent context
	ctx := context.Background()
	// FIXME: 476 - we should calculate the expirationHeight somehow (by asking
	// the chain for the last block through the RPC bus and calculating the
	// height)
	tx, err := t.proxy.Provider().NewStake(ctx, blsKey, req.Amount)
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

	log.
		WithField("amount", req.Amount).
		WithField("locktime", req.Locktime).
		Trace("Success creating a stake tx")

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
		Trace("Create a standard tx")

	ctx := context.Background()

	pb, err := DecodeAddressToPublicKey(req.Address)
	if err != nil {
		return nil, err
	}

	start := time.Now().UnixNano()

	tx, err := t.proxy.Provider().NewTransfer(ctx, req.Amount, pb)
	if err != nil {
		log.
			WithField("amount", req.Amount).
			WithField("address", string(req.Address)).
			Error("handleSendStandardTx, failed to create NewTransactionTx")
		return nil, err
	}

	d := (time.Now().UnixNano() - start) / (1000 * 1000)
	log.WithField("duration_ms", d).Debug("NewTransfer grpc call")

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
