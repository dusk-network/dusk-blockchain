package transactor

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

//nolint
type Transactor struct { // TODO: rename
	db database.DB
	eb *eventbus.EventBus

	// Passed to the consensus component startup
	// c                 *chainsync.Counter
	// acceptedBlockChan <-chan block.Block

	grpcClient rusk.RuskClient
}

// New Instantiate a new Transactor struct.
func New(eb *eventbus.EventBus, db database.DB, srv *grpc.Server, client rusk.RuskClient) *Transactor {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	t := &Transactor{
		db: db,
		eb: eb,
		// c:           counter,
		grpcClient: client,
	}

	if srv != nil {
		node.RegisterWalletServer(srv, t)
		node.RegisterTransactorServer(srv, t)
	}
	return t
}

func (t *Transactor) GetTxHistory(ctx context.Context, e *node.EmptyRequest) (*node.TxHistoryResponse, error) {
	return t.handleGetTxHistory()
}

// CreateWallet creates a new wallet from a password or seed
func (t *Transactor) CreateWallet(ctx context.Context, c *node.CreateRequest) (*node.LoadResponse, error) {
	return t.handleCreateWallet(c)
}

// LoadWallet from a password
func (t *Transactor) LoadWallet(ctx context.Context, l *node.LoadRequest) (*node.LoadResponse, error) {
	return t.handleLoadWallet(l)
}

// CreateFromSeed creates a wallet from a seed
func (t *Transactor) CreateFromSeed(ctx context.Context, c *node.CreateRequest) (*node.LoadResponse, error) {
	return t.handleCreateFromSeed(c)
}

func (t *Transactor) ClearWalletDatabase(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	return t.handleClearWalletDatabase()
}

func (t *Transactor) CallContract(ctx context.Context, c *node.CallContractRequest) (*node.TransactionResponse, error) {
	// TODO: implement
	return nil, nil
}

func (t *Transactor) Transfer(ctx context.Context, tr *node.TransferRequest) (*node.TransactionResponse, error) {
	return t.handleSendStandardTx(tr)
}

func (t *Transactor) Bid(ctx context.Context, c *node.BidRequest) (*node.TransactionResponse, error) {
	return t.handleSendBidTx(c)
}

func (t *Transactor) Stake(ctx context.Context, c *node.StakeRequest) (*node.TransactionResponse, error) {
	return t.handleSendStakeTx(c)
}

func (t *Transactor) GetWalletStatus(ctx context.Context, e *node.EmptyRequest) (*node.WalletStatusResponse, error) {
	return t.handleIsWalletLoaded()
}

func (t *Transactor) GetAddress(ctx context.Context, e *node.EmptyRequest) (*node.LoadResponse, error) {
	return t.handleAddress()
}

func (t *Transactor) GetBalance(ctx context.Context, e *node.EmptyRequest) (*node.BalanceResponse, error) {
	return t.handleBalance()
}
