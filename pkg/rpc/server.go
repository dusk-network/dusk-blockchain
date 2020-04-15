// This package represents the GRPC server exposing functions to interoperate
// with the node components as well as the wallet
package rpc

import (
	"context"
	"encoding/base64"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logger.WithFields(logger.Fields{"prefix": "grpc"})

// Ensure `nodeServer` implements `node.NodeServer`
var _ node.NodeServer = (*nodeServer)(nil)

// nodeServer is the gRPC entry-point for the DUSK node. nodeServer
// implements the dusk-protobuf/node.NodeServer interface.
type nodeServer struct {
	rpcBus *rpcbus.RPCBus
}

// SrvWrapper is a wrapper for the grpc server
type SrvWrapper struct {
	grpcServer *grpc.Server
}

// Shutdown the wrapper
func (r *SrvWrapper) Shutdown() {
	r.grpcServer.GracefulStop()
}

// StartgRPCServer starts the gRPC server for the node, to interact with
// the rust process. It only returns an error, as we want to keep the
// gRPC service running until the process is killed, thus we do not
// need to return the server itself.
func StartgRPCServer(rpcBus *rpcbus.RPCBus) (*SrvWrapper, error) {

	conf := config.Get().RPC
	l, err := net.Listen(conf.Network, conf.Address)
	if err != nil {
		return nil, err
	}

	// Build basic auth token, if configured
	if len(conf.User) > 0 && len(conf.Pass) > 0 {
		msg := conf.User + ":" + conf.Pass
		token = base64.StdEncoding.EncodeToString([]byte(msg))
	} else {
		if conf.Network != "unix" {
			log.WithError(err).Panicf("basic auth is disabled on %s network", conf.Network)
		}
	}

	// Add default interceptors to provide basic authentication and error logging
	// for both unary and stream RPC calls
	serverOpt := make([]grpc.ServerOption, 0)
	serverOpt = append(serverOpt, grpc.StreamInterceptor(streamInterceptor))
	serverOpt = append(serverOpt, grpc.UnaryInterceptor(unaryInterceptor))

	// Enable TLS if configured
	opt, tlsVer := loadTLSFiles(conf.EnableTLS, conf.CertFile, conf.KeyFile, conf.Network)
	if opt != nil {
		serverOpt = append(serverOpt, opt)
	}

	grpcServer := grpc.NewServer(serverOpt...)
	grpc.EnableTracing = false

	node.RegisterNodeServer(grpcServer, &nodeServer{rpcBus})
	wrapper := &SrvWrapper{grpcServer}

	// This function is blocking, so we run it in a goroutine
	go func() {
		log.WithField("net", conf.Network).
			WithField("addr", conf.Address).
			WithField("tls", tlsVer).Infof("gRPC HTTP server listening")

		if err := grpcServer.Serve(l); err != nil {
			log.WithError(err).Warn("Serve returned err")
		}
	}()

	return wrapper, nil
}

// SelectTx returns the transactions from the Mempool. It accepts a
// SelectRequest carrying either the ID of a specific transaction or the types
// of transactions as in "COINBASE", "BID", "STAKE", "STANDARD", "TIMELOCK", "CONTRACT"
func (n *nodeServer) SelectTx(ctx context.Context, req *node.SelectRequest) (*node.SelectResponse, error) {
	txs, err := n.rpcBus.Call(topics.GetMempoolView, rpcbus.NewRequest(req), 5*time.Second)
	if err != nil {
		return nil, err
	}

	return txs.(*node.SelectResponse), nil
}

//// CreateWallet creates a new wallet from a password or seed
//func (n *nodeServer) CreateWallet(ctx context.Context, c *node.CreateRequest) (*node.LoadResponse, error) {
//	resp, err := n.rpcBus.Call(topics.CreateWallet, rpcbus.NewRequest(c), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.LoadResponse), nil
//}

// LoadWallet from a password
//func (n *nodeServer) LoadWallet(ctx context.Context, l *node.LoadRequest) (*node.LoadResponse, error) {
//	resp, err := n.rpcBus.Call(topics.LoadWallet, rpcbus.NewRequest(l), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.LoadResponse), nil
//}
//
//// CreateFromSeed creates a wallet from a seed
//func (n *nodeServer) CreateFromSeed(ctx context.Context, c *node.CreateRequest) (*node.LoadResponse, error) {
//	resp, err := n.rpcBus.Call(topics.CreateFromSeed, rpcbus.NewRequest(c), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.LoadResponse), nil
//}
//
//func (n *nodeServer) ClearWalletDatabase(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
//	resp, err := n.rpcBus.Call(topics.ClearWalletDatabase, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.GenericResponse), nil
//}
//
//func (n *nodeServer) Transfer(ctx context.Context, t *node.TransferRequest) (*node.TransferResponse, error) {
//	resp, err := n.rpcBus.Call(topics.SendStandardTx, rpcbus.NewRequest(t), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.TransferResponse), nil
//}
//
//func (n *nodeServer) SendBid(ctx context.Context, c *node.ConsensusTxRequest) (*node.TransferResponse, error) {
//	resp, err := n.rpcBus.Call(topics.SendBidTx, rpcbus.NewRequest(c), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.TransferResponse), nil
//}
//
//func (n *nodeServer) SendStake(ctx context.Context, c *node.ConsensusTxRequest) (*node.TransferResponse, error) {
//	resp, err := n.rpcBus.Call(topics.SendStakeTx, rpcbus.NewRequest(c), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.TransferResponse), nil
//}
//
//func (n *nodeServer) AutomateConsensusTxs(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
//	resp, err := n.rpcBus.Call(topics.AutomateConsensusTxs, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.GenericResponse), nil
//}
//
//func (n *nodeServer) GetWalletStatus(ctx context.Context, e *node.EmptyRequest) (*node.WalletStatusResponse, error) {
//	resp, err := n.rpcBus.Call(topics.IsWalletLoaded, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.WalletStatusResponse), nil
//}
//
//func (n *nodeServer) GetAddress(ctx context.Context, e *node.EmptyRequest) (*node.LoadResponse, error) {
//	resp, err := n.rpcBus.Call(topics.GetAddress, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.LoadResponse), nil
//}
//
//func (n *nodeServer) GetSyncProgress(ctx context.Context, e *node.EmptyRequest) (*node.SyncProgressResponse, error) {
//	resp, err := n.rpcBus.Call(topics.GetSyncProgress, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.SyncProgressResponse), nil
//}
//
//func (n *nodeServer) GetBalance(ctx context.Context, e *node.EmptyRequest) (*node.BalanceResponse, error) {
//	resp, err := n.rpcBus.Call(topics.GetBalance, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.BalanceResponse), nil
//}
//
//func (n *nodeServer) GetUnconfirmedBalance(ctx context.Context, e *node.EmptyRequest) (*node.BalanceResponse, error) {
//	resp, err := n.rpcBus.Call(topics.GetUnconfirmedBalance, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.BalanceResponse), nil
//}
//
//func (n *nodeServer) GetTxHistory(ctx context.Context, e *node.EmptyRequest) (*node.TxHistoryResponse, error) {
//	resp, err := n.rpcBus.Call(topics.GetTxHistory, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.TxHistoryResponse), nil
//}

//TODO: un-comment it once gRPC EmptyRequest GenericResponse are restored
//TODO #363
//
//func (n *nodeServer) RebuildChain(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
//	resp, err := n.rpcBus.Call(topics.RebuildChain, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.GenericResponse), nil
//}
//
//func (n *nodeServer) StopProfile(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
//
//	resp, err := n.rpcBus.Call(topics.StopProfile, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		log.WithError(err).Warnln("StopProfile gRPC request failed", err)
//		return nil, err
//	}
//
//	return resp.(*node.GenericResponse), nil
//}
//
//func (n *nodeServer) StartProfile(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
//	resp, err := n.rpcBus.Call(topics.StartProfile, rpcbus.NewRequest(e), 5*time.Second)
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.(*node.GenericResponse), nil
//}

func loadTLSFiles(enable bool, certFile, keyFile, network string) (grpc.ServerOption, string) {

	tlsVersion := "disabled"
	if !enable {
		if network != "unix" {
			// Running gRPC over tcp would require TLS
			log.Warn("Running over insecure HTTP")
		}

		return nil, tlsVersion
	}

	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		// If TLS is explicitly enabled, any error here should cause panic
		log.WithError(err).Panic("could not enable TLS")
	}

	i := creds.Info()
	if i.SecurityProtocol == "ssl" {
		log.WithError(err).Panic("SSL is insecure")
	}

	recommendedVer := "1.3"
	if i.SecurityVersion != recommendedVer { //nolint
		log.Warnf("Recommended TLS version is %s", recommendedVer)
	}

	tlsVersion = i.SecurityVersion //nolint
	return grpc.Creds(creds), tlsVersion
}
