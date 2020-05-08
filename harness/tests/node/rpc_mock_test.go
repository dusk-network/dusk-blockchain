package node

import (
	"context"
	"fmt"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"

	"net"
	"testing"

	"google.golang.org/grpc"
)

func TestSayHello(t *testing.T) {
	s := grpc.NewServer()
	node.RegisterWalletServer(s, &node.WalletMock{})

	lis, _ := net.Listen("tcp", "127.0.0.1:5051")
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, "127.0.0.1:5051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := node.NewWalletClient(conn)

	callCtx, cancelCtx := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelCtx()

	resp, err := client.GetTxHistory(callCtx, &node.EmptyRequest{})
	if err != nil {
		t.Fatalf("SayHello failed: %v", err)
	}
	t.Logf("Response: %+v", resp)
}
