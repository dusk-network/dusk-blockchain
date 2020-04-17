package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/cmd/wallet/conf"
	"github.com/dusk-network/dusk-blockchain/cmd/wallet/prompt"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
)

func main() {
	defer handlePanic()

	config := conf.InitConfig()

	// Establish a gRPC connection with the node.
	_, _ = fmt.Fprintln(os.Stdout, "Wallet will establish a gRPC connection with the node.", config.RPC.Address)
	client := conf.NewNodeClient()

	if err := client.Connect(config.RPC); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// TODO: deferred functions are not run when os.Exit is called
	defer client.Close()

	// Inquire node about its wallet state, so we know which menu to open.
	resp, err := client.NodeClient.GetWalletStatus(context.Background(), &node.EmptyRequest{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// If we have no wallet loaded, we open the menu to load or
	// create one.
	if !resp.Loaded {
		if err := prompt.LoadMenu(client.NodeClient); err != nil {
			// If we get an error from `LoadMenu`, it means we lost
			// our connection to the node.
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	// Once loaded, we open the menu for wallet operations.
	if err := prompt.WalletMenu(client.NodeClient); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func handlePanic() {
	if r := recover(); r != nil {
		_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("%+v", r), "Application Wallet panic")
	}
	time.Sleep(time.Second * 1)
}
