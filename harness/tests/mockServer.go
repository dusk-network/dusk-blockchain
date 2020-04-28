package tests

import (
	"fmt"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"google.golang.org/grpc"
	"net"
)

func startMockServer() {
	s := grpc.NewServer()
	node.RegisterWalletServer(s, &node.WalletMock{})
	node.RegisterTransactorServer(s, &node.TransactorMock{})

	lis, _ := net.Listen("tcp", "127.0.0.1:8080")
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()
}
