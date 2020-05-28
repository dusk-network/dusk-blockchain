package server_test

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
	assert "github.com/stretchr/testify/require"
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
	// create the GRPC server here
	grpcSrv, err := server.SetupGRPC(conf)

	// panic in case of errors
	if err != nil {
		panic(err)
	}

	// get the server address from configuration
	go serve(conf.Network, conf.Address, grpcSrv)

	time.Sleep(5 * time.Second)
	// create the GRPC connection
	conn, err := grpc.Dial(
		conf.Address,
		grpc.WithInsecure(),
		grpc.WithContextDialer(getDialer("unix")),
	)
	if err != nil {
		panic(err)
	}

	// create the client
	authClient = client.NewClient(conn)
	// create the client interceptor to inject the session into another service
	// client
	interceptor, err := client.NewClientInterceptor(authClient, 0)
	if err != nil {
		panic(err)
	}

	wconn, err := grpc.Dial(
		conf.Address,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
	)

	if err != nil {
		panic(err)
	}

	walletClient = node.NewWalletClient(wconn)

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

func TestCreateSession(t *testing.T) {
	assert := assert.New(t)
	jwt, err := authClient.CreateSession()
	assert.NoError(err)
	assert.NotEmpty(jwt)
}
