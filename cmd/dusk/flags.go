package main

import (
	"github.com/urfave/cli"
)

var (
	// VerbosityFlag flag to set mode to verbose
	VerbosityFlag = cli.StringFlag{
		Name:  "verbosity",
		Usage: "verbosity",
	}
	// ConfigFlag flag to use configuration file
	ConfigFlag = cli.StringFlag{
		Name:  "config",
		Usage: "dusk.toml configuration file",
	}
	// DataDirFlag flag to set the data directory of the node
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the node",
	}
)

var (
	// CLIFlags flags usable in a CLI context
	CLIFlags = []cli.Flag{
		VerbosityFlag,
	}
	// GlobalFlags flags usable in a global context
	GlobalFlags = []cli.Flag{
		ConfigFlag,
		DataDirFlag,
	}
)
