// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/dusk-network/dusk-blockchain/cmd/wallet/conf"
	"github.com/dusk-network/dusk-blockchain/cmd/wallet/prompt"
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

	// Once connected, we open the menu for wallet operations.
	if err := prompt.WalletMenu(client); err != nil {
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
