// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactor

import (
	"context"

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
	bidChan   <-chan rpcbus.Request

	// Passed to the consensus component startup
	proxy transactions.Proxy

	w *wallet.Wallet
}

// New Instantiate a new Transactor struct.
func New(eb *eventbus.EventBus, rb *rpcbus.RPCBus, db database.DB, srv *grpc.Server, proxy transactions.Proxy, w *wallet.Wallet) (*Transactor, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	stakeChan := make(chan rpcbus.Request, 1)
	bidChan := make(chan rpcbus.Request, 1)

	t := &Transactor{
		db:        db,
		eb:        eb,
		rb:        rb,
		stakeChan: stakeChan,
		bidChan:   bidChan,
		proxy:     proxy,
		w:         w,
	}

	if srv != nil {
		node.RegisterWalletServer(srv, t)
		node.RegisterTransactorServer(srv, t)
	}

	if err := rb.Register(topics.SendStakeTx, stakeChan); err != nil {
		return nil, err
	}

	if err := rb.Register(topics.SendBidTx, bidChan); err != nil {
		return nil, err
	}

	go t.Listen()
	return t, nil
}

// Listen to the stake and bid channels and trigger a stake and bid transaction
// requests.
func (t *Transactor) Listen() {
	l := log.WithField("action", "listen")

	for {
		select {
		case r := <-t.stakeChan:
			req, ok := r.Params.(*node.StakeRequest)
			if !ok {
				continue
			}

			// QUESTION: should we return the hash of the transaction back to
			// the client?
			if _, err := t.Stake(context.Background(), req); err != nil {
				l.WithError(err).Error("error in creating a stake transaction")
			}

		case r := <-t.bidChan:
			req, ok := r.Params.(*node.BidRequest)
			if !ok {
				continue
			}

			if _, err := t.Bid(context.Background(), req); err != nil {
				l.WithError(err).Error("error in creating a bid transaction")
			}
		}
	}
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
	return t.handleSendBidTx(c)
}

// Stake will create a staking transaction.
func (t *Transactor) Stake(ctx context.Context, c *node.StakeRequest) (*node.TransactionResponse, error) {
	return t.handleSendStakeTx(c)
}

// GetAddress returns the address of the loaded wallet.
func (t *Transactor) GetAddress(ctx context.Context, e *node.EmptyRequest) (*node.LoadResponse, error) {
	return t.handleAddress()
}

// GetBalance returns the balance of the loaded wallet.
func (t *Transactor) GetBalance(ctx context.Context, e *node.EmptyRequest) (*node.BalanceResponse, error) {
	return t.handleBalance()
}
