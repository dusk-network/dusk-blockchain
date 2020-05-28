package main

import (
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/cmd/utils/metrics"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Dusk Exporter CMD"
	app.Usage = "The Dusk Exporter command line interface"

	app.Commands = []cli.Command{
		metricsCMD,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	gqlPortFlag = cli.IntFlag{
		Name:  "gqlport",
		Usage: "GQL PORT , eg: --gqlport=9503",
		Value: 9503,
	}

	nodePortFlag = cli.IntFlag{
		Name:  "nodeport",
		Usage: "Dusk node PORT , eg: --nodeport=9000",
		Value: 9000,
	}

	hostnameFlag = cli.StringFlag{
		Name:  "hostname",
		Usage: "Dusk hostname , eg: --hostname=127.0.0.1",
		Value: "127.0.0.1",
	}

	portFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Exporter probe port , eg: --port=9099",
		Value: 9099,
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
