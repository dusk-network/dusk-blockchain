package server_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
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

var authClient *client.AuthClient
var walletClient node.WalletClient

func init() {
	log.SetLevel(log.ErrorLevel)
}

var address = "/tmp/dusk-grpc-test01.sock"

func getDialer(proto string) func(context.Context, string) (net.Conn, error) {
	d := &net.Dialer{}
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return d.DialContext(ctx, proto, addr)
	}
}

func TestMain(m *testing.M) {
	conf := server.Setup{Network: "unix", Address: address}
	conf.RequireSession = true
	// create the GRPC server here
	grpcSrv, err := server.SetupGRPC(conf)

	// panic in case of errors
	if err != nil {
		panic(err)
	}

	// get the server address from configuration
	go serve(conf.Network, conf.Address, grpcSrv)

	// GRPC client bootstrap
	time.Sleep(200 * time.Millisecond)
	// create the client
	pk, sk, _ := ed25519.GenerateKey(rand.Reader)
	// injecting the interceptor
	interceptor := client.NewClientInterceptor(pk, sk)

	// create the GRPC connection
	conn, err := grpc.Dial(
		conf.Address,
		grpc.WithInsecure(),
		grpc.WithContextDialer(getDialer("unix")),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
	)

	if err != nil {
		panic(err)
	}

	// authClient performs the session calls
	authClient = client.NewClient(conn, pk, sk)
	// walletClient performs the wallet calls. It reuses the connection from
	// the authClient
	walletClient = node.NewWalletClient(conn)

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
