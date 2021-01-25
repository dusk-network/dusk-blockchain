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

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	log *logrus.Entry
	app = cli.NewApp()
)

func initLog() {
	log = logrus.WithFields(logrus.Fields{
		"app":    "voucher",
		"prefix": "main",
	})
}

func init() {
	initLog()

	app.Action = action
	app.Copyright = "Copyright (c) 2020 DUSK"
	app.Name = "voucher"
	app.Usage = "Official Voucher command-line interface"
	app.Author = "DUSK 2020"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{}
	app.Flags = append(app.Flags, CLIFlags...)
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
		log.WithError(fmt.Errorf("%+v", r)).Errorln("Application Voucher panic")
	}

	time.Sleep(time.Second * 1)
}
