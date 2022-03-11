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

	"github.com/Masterminds/semver"
	"github.com/dusk-network/dusk-blockchain/cmd/dusk/genesis"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var app = cli.NewApp()

func initLog() {
	log = logrus.WithFields(logrus.Fields{
		"app":    "dusk",
		"prefix": "main",
	})
}

func init() {
	initLog()

	app.Action = action
	app.Copyright = "Copyright (c) 2020 DUSK"
	app.Name = "dusk"
	app.Usage = "Official Dusk command-line interface"
	app.Author = "DUSK 2020"
	app.Version = semver.MustParse(cfg.NodeVersion).String()
	app.Commands = []cli.Command{
		{
			Name:    "genesis",
			Aliases: []string{"g"},
			Usage:   "serializes the genesis block and prints it",
			Action:  genesis.Action,
		},
	}
	app.Flags = append(app.Flags, CLIFlags...)
	app.Flags = append(app.Flags, GlobalFlags...)
}

func main() {
	defer handlePanic()

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handlePanic() {
	if r := recover(); r != nil {
		log.WithError(fmt.Errorf("%+v", r)).Errorln("Application panic")
	}

	time.Sleep(time.Second * 1)
}
