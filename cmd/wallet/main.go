package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/dusk-network/dusk-blockchain/cmd/wallet/conf"
	"github.com/dusk-network/dusk-blockchain/cmd/wallet/prompt"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
)

var (
	configPathFlag = cli.StringFlag{
		Name:  "configpath",
		Usage: "dusk toml path , eg: --configpath=/tmp/localnet-317173610/node-9000",
		Value: "",
	}
)

func main() {
	app := cli.NewApp()
	app.Usage = "The Dusk Wallet command line interface"
	app.Action = walletAction
	app.Flags = []cli.Flag{
		configPathFlag,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func walletAction(ctx *cli.Context) error {
	defer handlePanic()

	configPath := ctx.String(configPathFlag.Name)
	config := conf.InitConfig(configPath)

	// Establish a gRPC connection with the node.
	_, _ = fmt.Fprintln(os.Stdout, "Wallet will establish a gRPC connection with the node.", config.RPC.Address)
	client := conf.NewNodeClient()

	if err := client.Connect(config.RPC); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}
	// TODO: deferred functions are not run when os.Exit is called
	defer client.Close()

	// Inquire node about its wallet state, so we know which menu to open.
	resp, err := client.NodeClient.GetWalletStatus(context.Background(), &node.EmptyRequest{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}

	// If we have no wallet loaded, we open the menu to load or
	// create one.
	if !resp.Loaded {
		if err := prompt.LoadMenu(client.NodeClient); err != nil {
			// If we get an error from `LoadMenu`, it means we lost
			// our connection to the node.
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			return err
		}
	}

	// Once loaded, we open the menu for wallet operations.
	if err := prompt.WalletMenu(client.NodeClient); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		return err
	}
	return nil
}

func handlePanic() {
	if r := recover(); r != nil {
		_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("%+v", r), "Application Wallet panic")
	}
	time.Sleep(time.Second * 1)
}
