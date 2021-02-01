// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/mock"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/transactions"
	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/metrics"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Dusk Exporter CMD"
	app.Usage = "The Dusk Exporter command line interface"

	app.Commands = []cli.Command{
		metricsCMD,
		transactionsCMD,
		mockCMD,
		mockRUSKCMD,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	grpcHostFlag = cli.StringFlag{
		Name:  "grpchost",
		Usage: "gRPC HOST , eg: --grpchost=127.0.0.1:9001",
		Value: "127.0.0.1:9001",
	}

	amountFlag = cli.Uint64Flag{
		Name:  "amount",
		Usage: "amount , eg: --amount=1",
		Value: 10,
	}

	lockTimeFlag = cli.Uint64Flag{
		Name:  "locktime",
		Usage: "locktime , eg: --locktime=1",
		Value: 10,
	}

	txtypeFlag = cli.StringFlag{
		Name:  "txtype",
		Usage: "Dusk hostname , eg: --txtype=consensus",
		Value: "consensus",
	}

	addressFlag = cli.StringFlag{
		Name:  "address",
		Usage: "Dusk address , eg: --address=self",
		Value: "self",
	}

	gqlPortFlag = cli.IntFlag{
		Name:  "gqlport",
		Usage: "GQL PORT , eg: --gqlport=9500",
		Value: 9500,
	}

	nodePortFlag = cli.IntFlag{
		Name:  "nodeport",
		Usage: "Dusk node PORT , eg: --nodeport=9000",
		Value: 9000,
	}

	nodeAPIPortFlag = cli.IntFlag{
		Name:  "nodeapiport",
		Usage: "Dusk API node PORT , eg: --nodeapiport=9490",
		Value: 9490,
	}

	portFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Exporter probe port , eg: --port=9099",
		Value: 9099,
	}
	hostnameFlag = cli.StringFlag{
		Name:  "hostname",
		Usage: "Dusk hostname , eg: --hostname=127.0.0.1",
		Value: "127.0.0.1",
	}

	grpcMockHostFlag = cli.StringFlag{
		Name:  "grpcmockhost",
		Usage: "gRPC HOST , eg: --grpcmockhost=127.0.0.1:9191",
		Value: "127.0.0.1:9191",
	}

	ruskNetworkFlag = cli.StringFlag{
		Name:  "rusknetwork",
		Usage: "Dusk RUSK network , eg: --rusknetwork=tcp",
		Value: "tcp",
	}

	ruskAddressFlag = cli.StringFlag{
		Name:  "ruskaddress",
		Usage: "Dusk hostname , eg: --hostname=127.0.0.1:10000",
		Value: "127.0.0.1:10000",
	}

	walletStoreFlag = cli.StringFlag{
		Name:  "walletstore",
		Usage: "Dusk hostname , eg: --walletstore=/tmp/localnet-137601832/node-9003/walletDB/",
		Value: "/tmp/localnet-137601832/node-9003/walletDB/",
	}
	walletFileFlag = cli.StringFlag{
		Name:  "walletfile",
		Usage: "Dusk hostname , eg: --walletfile=./data/wallet-9000.dat",
		Value: "./devnet-wallets/wallet0.dat",
	}

	// XXX: This seems unused now. We should figure out if that's alright,
	// or if we need to update some of the logic on wallet creation.
	//nolint
	walletPasswordFlag = cli.StringFlag{
		Name:  "walletpassword",
		Usage: "Dusk Wallet Password, eg: --walletpassword=password",
		Value: "password",
	}

	metricsCMD = cli.Command{
		Name:      "metrics",
		Usage:     "expose a metrics endpoint",
		Action:    metricsAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			gqlPortFlag,
			nodePortFlag,
			nodeAPIPortFlag,
			portFlag,
			hostnameFlag,
		},
		Description: `Expose a Dusk metrics endpoint to be consumed by Prometheus`,
	}

	transactionsCMD = cli.Command{
		Name:      "transactions",
		Usage:     "execute transactions (consensus, stake, transfer)",
		Action:    transactionsAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			txtypeFlag,
			grpcHostFlag,
			amountFlag,
			lockTimeFlag,
			addressFlag,
		},
		Description: `Execute/Query transactions for a Dusk node`,
	}

	mockCMD = cli.Command{
		Name:      "mock",
		Usage:     "execute a mock server",
		Action:    mockAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			grpcMockHostFlag,
		},
		Description: `Execute/Query transactions for a Dusk node`,
	}

	mockRUSKCMD = cli.Command{
		Name:      "mockrusk",
		Usage:     "execute a mock rusk server",
		Action:    mockRuskAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			ruskNetworkFlag,
			ruskAddressFlag,
			walletStoreFlag,
			walletFileFlag,
		},
		Description: `Execute/Query transactions for a Dusk node`,
	}
)

// metricsAction will expose the metrics endpoint.
func metricsAction(ctx *cli.Context) error {
	gqlPort := ctx.Int(gqlPortFlag.Name)
	nodePort := ctx.Int(nodePortFlag.Name)
	nodeAPIPort := ctx.Int(nodeAPIPortFlag.Name)
	port := ctx.Int(portFlag.Name)
	hostname := ctx.String(hostnameFlag.Name)

	metrics.RunMetrics(gqlPort, nodePort, nodeAPIPort, port, hostname)

	return nil
}

// transactionsAction will expose the metrics endpoint.
func transactionsAction(ctx *cli.Context) error {
	grpcHost := ctx.String(grpcHostFlag.Name)
	amount := ctx.Uint64(amountFlag.Name)
	lockTime := ctx.Uint64(lockTimeFlag.Name)
	txtype := ctx.String(txtypeFlag.Name)
	address := ctx.String(addressFlag.Name)

	transfer := transactions.Transaction{
		Amount: amount, LockTime: lockTime,
		TXtype: txtype, Address: address,
	}

	log.WithField("transfer", transfer).
		Info("transactions Action started")

	transferResponse, err := transactions.RunTransactions(
		grpcHost,
		transfer,
	)
	if err != nil {
		return err
	}

	txHash := hex.EncodeToString(transferResponse.Hash)

	log.WithField("txHash", txHash).
		Info("transactions Action completed")

	return nil
}

func mockAction(ctx *cli.Context) error {
	grpcMockHost := ctx.String(grpcMockHostFlag.Name)

	err := mock.RunMock(grpcMockHost)
	return err
}

func mockRuskAction(ctx *cli.Context) error {
	ruskNetwork := ctx.String(ruskNetworkFlag.Name)
	ruskAddress := ctx.String(ruskAddressFlag.Name)

	walletStore := ctx.String(walletStoreFlag.Name)
	walletFile := ctx.String(walletFileFlag.Name)

	err := mock.RunRUSKMock(ruskNetwork, ruskAddress, walletStore, walletFile)
	return err
}
