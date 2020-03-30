package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"time"
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
	app.Version = "0.0.1"
	app.Commands = []cli.Command{}
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
		log.WithError(fmt.Errorf("%+v", r)).Error(fmt.Sprintf("Application panic"))
	}
	time.Sleep(time.Second * 1)
}
