package server_test

import (
	"os"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"google.golang.org/grpc"
)

var client *AuthClient

func TestMain(m *testing.M) {
	// create the GRPC server here
	srv, err := SetupGRPCServer()
	// panic in case of errors
	if err != nil {
		panic(err)
	}

	// get the server address from configuration
	conf := config.Get().RPC
	// create the GRPC connection
	conn := grpc.Dial(conf.Address, grpc.WithInsecure())
	// create the client
	client = NewClient(conn)
	// run the tests
	res := m.Run()
	// done
	os.Exit(res)
}

func TestCreateSession(t *testing.T) {

}
