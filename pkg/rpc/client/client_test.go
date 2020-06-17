package client_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/server"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// WWalletSrvMock is a mockup structure used to test the session client and server
type WalletSrvMock struct{}

// GetTxHistory will return a subset of the transactions that were sent and received.
func (t *WalletSrvMock) GetTxHistory(ctx context.Context, e *node.EmptyRequest) (*node.TxHistoryResponse, error) {
	return &node.TxHistoryResponse{}, nil
}

// CreateWallet creates a new wallet from a password or seed
func (t *WalletSrvMock) CreateWallet(ctx context.Context, c *node.CreateRequest) (*node.LoadResponse, error) {
	return &node.LoadResponse{}, nil
}

// LoadWallet from a password
func (t *WalletSrvMock) LoadWallet(ctx context.Context, l *node.LoadRequest) (*node.LoadResponse, error) {
	return &node.LoadResponse{}, nil
}

// CreateFromSeed creates a wallet from a seed
func (t *WalletSrvMock) CreateFromSeed(ctx context.Context, c *node.CreateRequest) (*node.LoadResponse, error) {
	return &node.LoadResponse{}, nil
}

// ClearWalletDatabase clears the wallet database, containing the unspent outputs.
func (t *WalletSrvMock) ClearWalletDatabase(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	return &node.GenericResponse{}, nil
}

// CallContract will create a transaction that calls a smart contract.
func (t *WalletSrvMock) CallContract(ctx context.Context, c *node.CallContractRequest) (*node.TransactionResponse, error) {
	return &node.TransactionResponse{}, nil
}

// Transfer will create a normal transaction, transferring DUSK.
func (t *WalletSrvMock) Transfer(ctx context.Context, tr *node.TransferRequest) (*node.TransactionResponse, error) {
	return &node.TransactionResponse{}, nil
}

// Bid will create a bidding transaction.
func (t *WalletSrvMock) Bid(ctx context.Context, c *node.BidRequest) (*node.TransactionResponse, error) {
	return &node.TransactionResponse{}, nil
}

// Stake will create a staking transaction.
func (t *WalletSrvMock) Stake(ctx context.Context, c *node.StakeRequest) (*node.TransactionResponse, error) {
	return &node.TransactionResponse{}, nil
}

// GetWalletStatus returns whether or not the wallet is currently loaded.
func (t *WalletSrvMock) GetWalletStatus(ctx context.Context, e *node.EmptyRequest) (*node.WalletStatusResponse, error) {
	return &node.WalletStatusResponse{}, nil
}

// GetAddress returns the address of the loaded wallet.
func (t *WalletSrvMock) GetAddress(ctx context.Context, e *node.EmptyRequest) (*node.LoadResponse, error) {
	return &node.LoadResponse{}, nil
}

// GetBalance returns the balance of the loaded wallet.
func (t *WalletSrvMock) GetBalance(ctx context.Context, e *node.EmptyRequest) (*node.BalanceResponse, error) {
	return &node.BalanceResponse{}, nil
}

func init() {
	log.SetLevel(log.ErrorLevel)
}

var address = "/tmp/dusk-grpc-test01.sock"
var nodeClient *client.NodeClient

func TestMain(m *testing.M) {
	conf := server.Setup{Network: "unix", Address: address}
	// create the GRPC server here
	grpcSrv, err := server.SetupGRPC(conf)

	// panic in case of errors
	if err != nil {
		panic(err)
	}

	// register wallet mock server to be able to test the session
	// the mock is replying with no error and an empty response
	node.RegisterWalletServer(grpcSrv, &WalletSrvMock{})

	// get the server address from configuration
	go serve(conf.Network, conf.Address, grpcSrv)

	// GRPC client bootstrap
	time.Sleep(200 * time.Millisecond)
	// create the client
	nodeClient = client.New("unix", address)

	// run the tests
	res := m.Run()

	_ = os.Remove(address)

	// done
	os.Exit(res)
}

func serve(network, addr string, srv *grpc.Server) {
	l, lerr := net.Listen(network, addr)
	if lerr != nil {
		panic(lerr)
	}

	if serr := srv.Serve(l); serr != nil {
		panic(serr)
	}
}
