// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import "github.com/urfave/cli"

var (
	// LogLevelFlag flag to set log level.
	LogLevelFlag = cli.StringFlag{
		Name:  "loglevel",
		Usage: "log level, eg: (warn, error, fatal, panic)",
		Value: "trace",
	}
	portFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Exporter probe port , eg: --port=9099",
		Value: 8081,
	}
	hostnameFlag = cli.StringFlag{
		Name:  "hostname",
		Usage: "Dusk hostname , eg: --hostname=127.0.0.1",
		Value: "127.0.0.1",
	}
)

// CLIFlags flags usable in a CLI context.
var CLIFlags = []cli.Flag{
	LogLevelFlag,
	portFlag,
	hostnameFlag,
}
