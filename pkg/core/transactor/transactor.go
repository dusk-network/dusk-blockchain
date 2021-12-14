// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactor

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"google.golang.org/grpc"
)

// Transactor is the implementation of both the Waller and the Transactor GRPC servers.
type Transactor struct {
	db database.DB
	eb *eventbus.EventBus
	rb *rpcbus.RPCBus

	// RPCBus channels
	stakeChan <-chan rpcbus.Request

	// Passed to the consensus component startup
	proxy transactions.Proxy

	// Used to retrieve the node's sync progress
	getSyncProgress func() float64

	w *wallet.Wallet
}

// New Instantiate a new Transactor struct.
func New(eb *eventbus.EventBus, rb *rpcbus.RPCBus, db database.DB, srv *grpc.Server, proxy transactions.Proxy, w *wallet.Wallet, getSyncProgress func() float64) (*Transactor, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	stakeChan := make(chan rpcbus.Request, 1)

	t := &Transactor{
		db:              db,
		eb:              eb,
		rb:              rb,
		stakeChan:       stakeChan,
		proxy:           proxy,
		w:               w,
		getSyncProgress: getSyncProgress,
	}

	if srv != nil {
		node.RegisterWalletServer(srv, t)
		node.RegisterTransactorServer(srv, t)
	}

	if err := rb.Register(topics.SendStakeTx, stakeChan); err != nil {
		return nil, err
	}

	go t.Listen()
	return t, nil
}

// Listen to the stake and bid channels and trigger a stake and bid transaction
// requests.
func (t *Transactor) Listen() {
	l := log.WithField("action", "listen")

	for r := range t.stakeChan {
		req, ok := r.Params.(*node.StakeRequest)
		if !ok {
			continue
		}

		var hash *bytes.Buffer

		resp, err := t.Stake(context.Background(), req)
		if err != nil {
			l.WithError(err).Error("error in creating a stake transaction")
		} else {
			hash = bytes.NewBuffer(resp.GetHash())
		}
		r.RespChan <- rpcbus.Response{Resp: hash, Err: err}
	}
}

func (t *Transactor) canStake() bool {
	progress := t.getSyncProgress()
	if progress == 100 {
		return true
	}

	log.WithField("progress", progress).Debug("could not send stake tx - node is not synced")

	// Check for our sync progress three more times. If we still can't stake after
	// the third check, it's better to just return false, and give the Transactor
	// control back over the Listen goroutine.
	interval := 5 * time.Second

	for i := 0; i < 3; i++ {
		time.Sleep(interval * time.Duration(i+1))

		progress = t.getSyncProgress()
		if progress == 100 {
			return true
		}

		log.WithField("progress", progress).Debug("could not send stake tx - node is not synced")
	}

	return false
}

// GetTxHistory will return a subset of the transactions that were sent and received.
func (t *Transactor) GetTxHistory(ctx context.Context, e *node.EmptyRequest) (*node.TxHistoryResponse, error) {
	return t.handleGetTxHistory()
}

// ClearWalletDatabase clears the wallet database, containing the unspent outputs.
func (t *Transactor) ClearWalletDatabase(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	return t.handleClearWalletDatabase()
}

// CallContract will create a transaction that calls a smart contract.
func (t *Transactor) CallContract(ctx context.Context, c *node.CallContractRequest) (*node.TransactionResponse, error) {
	return t.handleSendContract(c)
}

// Transfer will create a normal transaction, transferring DUSK.
func (t *Transactor) Transfer(ctx context.Context, tr *node.TransferRequest) (*node.TransactionResponse, error) {
	return t.handleSendStandardTx(tr)
}

// Bid will create a bidding transaction.
func (t *Transactor) Bid(ctx context.Context, c *node.BidRequest) (*node.TransactionResponse, error) {
	return nil, nil
}

// Stake will create a staking transaction.
func (t *Transactor) Stake(ctx context.Context, c *node.StakeRequest) (*node.TransactionResponse, error) {
	// Are we synced?
	if !t.canStake() {
		return nil, errors.New("node is not synced")
	}

	return t.handleSendStakeTx(c)
}

// GetAddress returns the address of the loaded wallet.
func (t *Transactor) GetAddress(ctx context.Context, e *node.EmptyRequest) (*node.LoadResponse, error) {
	return t.handleAddress()
}

// GetBalance returns the balance of the loaded wallet.
func (t *Transactor) GetBalance(ctx context.Context, e *node.EmptyRequest) (*node.BalanceResponse, error) {
	// NOT IMPL
	return &node.BalanceResponse{}, nil
}
