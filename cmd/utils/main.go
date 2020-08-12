package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/walletutils"

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
		walletUtilsCMD,
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

	walletCMDFlag = cli.StringFlag{
		Name:  "walletcmd",
		Usage: "Dusk WalletCmd , eg: --walletcmd=loadwallet",
		Value: "loadwallet",
	}

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

	walletUtilsCMD = cli.Command{
		Name:      "walletutils",
		Usage:     "execute cmd to a Dusk wallet",
		Action:    walletAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			grpcHostFlag,
			walletCMDFlag,
			walletPasswordFlag,
		},
		Description: `Execute/Query a Dusk wallet`,
	}
)

// metricsAction will expose the metrics endpoint
func metricsAction(ctx *cli.Context) error {

	gqlPort := ctx.Int(gqlPortFlag.Name)
	nodePort := ctx.Int(nodePortFlag.Name)
	port := ctx.Int(portFlag.Name)
	hostname := ctx.String(hostnameFlag.Name)

	metrics.RunMetrics(gqlPort, nodePort, port, hostname)

	return nil
}

// transactionsAction will expose the metrics endpoint
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

func walletAction(ctx *cli.Context) error {

	grpcHost := ctx.String(grpcHostFlag.Name)
	walletCMD := ctx.String(walletCMDFlag.Name)
	walletPassword := ctx.String(walletPasswordFlag.Name)

	log.WithField("walletCMD", walletCMD).
		Info("wallet Action started")

	resp, err := walletutils.RunWallet(grpcHost, walletCMD, walletPassword)

	log.WithField("resp", resp).
		Info("wallet Action completed")
	return err
}
