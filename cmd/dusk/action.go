// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/logging"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	log     *logrus.Entry
	config  string
	datadir string
)

func action(ctx *cli.Context) error {
	// check arguments
	if arguments := ctx.Args(); len(arguments) > 0 {
		return fmt.Errorf("failed to read command argument: %q", arguments[0])
	}

	if datadir = ctx.GlobalString(DataDirFlag.Name); datadir != "" {
		datadir = "~/./dusk"
	}

	if config = ctx.GlobalString(ConfigFlag.Name); config != "" {
		config = "dusk"
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Loading all node configurations. Fail-fast if critical error occurs
	err := cfg.Load(config, nil, nil)
	if err != nil {
		log.WithError(err).Fatal("Could not load config ")
	}

	rand.Seed(time.Now().UnixNano())

	// Set up logging.
	// Any subsystem should be initialized after config and logger loading
	var logFile *os.File

	output := cfg.Get().Logger.Output
	if output != "stdout" {
		logFile, err = os.Create(output + ".log")
		if err != nil {
			log.Panic(err)
		}

		defer func() {
			_ = logFile.Close()
		}()
	} else {
		logFile = os.Stdout
	}

	if cfg.Get().Logger.Format == "json" {
		log.Trace("Dusk log format set to JSON.")
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	logging.InitLog(logFile)

	log.WithField("file", cfg.Get().UsedConfigFile).Info("Loaded config file")
	log.WithField("network", cfg.Get().General.Network).Info("Selected network")

	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup()

	// Setting up profiling tools, if enabled
	s := setupProfiles(srv.rpcBus)

	log.Info("initialization complete")

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt

	s.Close()
	srv.Close()

	// Graceful shutdown of listening components
	msg := message.New(topics.Quit, bytes.Buffer{})
	errList := srv.eventBus.Publish(topics.Quit, msg)
	diagnostics.LogPublishErrors("dusk/action.go, topics.Quit", errList)

	time.Sleep(3 * time.Second)

	log.WithField("prefix", "main").Info("Terminated")

	return nil
}

func setupProfiles(r *rpcbus.RPCBus) *diagnostics.ProfileSet {
	s := diagnostics.NewProfileSet()
	profiles := cfg.Get().Profile

	// Expecting an array of profiles.
	// Add empty [[profile]] to enable the listener
	if len(profiles) > 0 {
		for _, i := range profiles {
			if len(i.Name) == 0 {
				continue
			}

			p := diagnostics.NewProfile(i.Name, i.Interval, i.Duration, i.Start)
			if err := s.Spawn(p); err != nil {
				log.WithError(err).Panic("Profiling task error")
			}
		}

		go s.Listen(r)
	}

	return &s
}
