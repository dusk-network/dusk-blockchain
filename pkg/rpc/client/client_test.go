package client_test

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/server"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

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
