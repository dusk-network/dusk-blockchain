package tests

import (
	"fmt"
	"net"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

// StartMockServer will start a mock server
func StartMockServer(address string) {
	s := grpc.NewServer()
	rusk.RegisterRuskServer(s, &rusk.RuskMock{})
	node.RegisterWalletServer(s, &node.WalletMock{})
	node.RegisterTransactorServer(s, &node.TransactorMock{})

	lis, _ := net.Listen("tcp", address)
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()
}
