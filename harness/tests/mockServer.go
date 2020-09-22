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
	rusk.RegisterStateServer(s, &rusk.StateMock{})
	rusk.RegisterKeysServer(s, &rusk.KeysMock{})
	node.RegisterWalletServer(s, &node.WalletMock{})
	node.RegisterTransactorServer(s, &node.TransactorMock{})

	lis, _ := net.Listen("tcp", address)
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()
}
