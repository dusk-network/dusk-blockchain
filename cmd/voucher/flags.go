package main

import "github.com/urfave/cli"

var (
	// LogLevelFlag flag to set log level
	LogLevelFlag = cli.StringFlag{
		Name:  "loglevel",
		Usage: "log level, eg: (warn, error, fatal, panic)",
	}
)

var (
	// CLIFlags flags usable in a CLI context
	CLIFlags = []cli.Flag{
		LogLevelFlag,
	}
)
