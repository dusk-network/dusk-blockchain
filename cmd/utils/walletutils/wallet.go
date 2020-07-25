package walletutils

import (
	"context"
	"errors"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"

	"time"

	"google.golang.org/grpc"
)

// RunWallet will run cmds against a dusk wallet
func RunWallet(grpcHost, walletCMD, walletPassword string) (*node.LoadResponse, error) {

	// Set up a connection to the server.
	conn, err := grpc.Dial(grpcHost, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := node.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if walletCMD == "loadwallet" {

		req := node.LoadRequest{Password: walletPassword}
		resp, err := client.LoadWallet(ctx, &req)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	return nil, errors.New("not yet implemented")
}
