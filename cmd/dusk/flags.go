package main

import (
	"github.com/urfave/cli"
)

var (
	VerbosityFlag = cli.StringFlag{
		Name:  "verbosity",
		Usage: "verbosity",
	}
	ConfigFlag = cli.StringFlag{
		Name:  "config",
		Usage: "dusk.toml configuration file",
	}
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the node",
	}
)

var (
	CLIFlags = []cli.Flag{
		VerbosityFlag,
	}
	GlobalFlags = []cli.Flag{
		ConfigFlag,
		DataDirFlag,
	}
)
