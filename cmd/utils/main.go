// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/grpcclient"
	"github.com/dusk-network/dusk-blockchain/cmd/utils/mock"
	"github.com/dusk-network/dusk-blockchain/cmd/utils/tps"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/metrics"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Dusk Exporter CMD"
	app.Usage = "The Dusk Exporter command line interface"

	app.Commands = []cli.Command{
		metricsCMD,
		mockCMD,
		setConfigCMD,
		tpsCMD,
		automateCMD,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	grpcAddressFlag = cli.StringFlag{
		Name:  "grpcaddr",
		Usage: "gRPC UNIX or TCP address , eg: --grpcaddr=127.0.0.1:9001 or --grpcaddr=unix:///var/dusk-grpc.sock",
		Value: "127.0.0.1:9001",
	}

	amountFlag = cli.Uint64Flag{
		Name:  "amount",
		Usage: "amount , eg: --amount=1",
		Value: 10,
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

	configNameFlag = cli.StringFlag{
		Name:  "configname",
		Usage: "Config ID from dusk.toml, eg: logger.level",
		Value: "",
	}

	configValueFlag = cli.StringFlag{
		Name:  "configvalue",
		Usage: "New Config value, eg: info",
		Value: "",
	}

	delayFlag = cli.IntFlag{
		Name:  "delay",
		Usage: "Set delay between sending of transactions in ms",
		Value: 0,
	}

	sendStakeTimeoutFlag = cli.IntFlag{
		Name:  "sendstaketimeout",
		Usage: "timeout for sending a stake request to Provisioner client",
		Value: 5,
	}

	sendBidTimeoutFlag = cli.IntFlag{
		Name:  "sendbidtimeout",
		Usage: "timeout for sending a bid request to BlockGenerator client",
		Value: 5,
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

	// setconfig command
	// Example ./bin/utils setconfig --grpcaddr="unix:///tmp/dusk-node/dusk-grpc.sock" --configname="logger.level" --configvalue="info".
	// 	     ./bin/utils setconfig --grpcaddr="127.0.0.1:10506" --configname="logger.level" --configvalue="trace".
	setConfigCMD = cli.Command{
		Name:      "setconfig",
		Usage:     "set config in run-time",
		Action:    setConfigAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			grpcAddressFlag,
			configNameFlag,
			configValueFlag,
		},
		Description: `Modify a specific dusk config in run-time`,
	}

	tpsCMD = cli.Command{
		Name:      "tps",
		Usage:     "attach to a node for continuous transaction spamming",
		Action:    tpsAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			grpcAddressFlag,
			delayFlag,
			amountFlag,
		},
		Description: `Send transactions from the given node until process exits`,
	}

	automateCMD = cli.Command{
		Name:      "automate",
		Usage:     "automate the sending of stakes and bids in a node",
		Action:    automateAction,
		ArgsUsage: "",
		Flags: []cli.Flag{
			grpcAddressFlag,
			sendStakeTimeoutFlag,
			sendBidTimeoutFlag,
		},
		Description: `Automate consensus participation of a node until the process exits`,
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

func mockAction(ctx *cli.Context) error {
	grpcMockHost := ctx.String(grpcMockHostFlag.Name)

	err := mock.RunMock(grpcMockHost)
	return err
}

func setConfigAction(ctx *cli.Context) error {
	req := grpcclient.SetConfigReq{
		Name:     ctx.String(configNameFlag.Name),
		NewValue: ctx.String(configValueFlag.Name),
	}

	req.Address = ctx.String(grpcAddressFlag.Name)
	return grpcclient.TrySetConfig(req)
}

func tpsAction(ctx *cli.Context) error {
	addr := ctx.String(grpcAddressFlag.Name)
	delay := ctx.Int(delayFlag.Name)
	amount := ctx.Uint64(amountFlag.Name)

	return tps.StartSpamming(addr, delay, amount)
}

func automateAction(ctx *cli.Context) error {
	address := ctx.String(grpcAddressFlag.Name)
	sendStakeTimeout := ctx.Int(sendStakeTimeoutFlag.Name)
	return grpcclient.AutomateStakes(address, sendStakeTimeout)
}
