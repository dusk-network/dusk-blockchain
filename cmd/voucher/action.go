// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"os"
	"os/signal"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/cmd/voucher/challenger"
	"github.com/dusk-network/dusk-blockchain/cmd/voucher/node"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	logger "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func action(ctx *cli.Context) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	eb := eventbus.New()

	if logLevel := ctx.GlobalString(LogLevelFlag.Name); logLevel != "" {
		log.WithField("logLevel", logLevel).Info("will configure log level")

		var err error

		log.Level, err = logger.ParseLevel(logLevel)
		if err != nil {
			log.WithError(err).Fatal("could not parse logLevel")
		}
	}

	store := node.NewStore()
	challenger := challenger.New(store)
	processor := peer.NewMessageProcessor(eb)

	processor.Register(topics.Response, challenger.ProcessResponse)
	processor.Register(topics.GetAddrs, store.DumpNodes)
	processor.Register(topics.Ping, responding.ProcessPing)
	processor.Register(topics.Pong, responding.ProcessPong)

	port := ctx.Int(portFlag.Name)
	c := peer.NewConnector(eb, protocol.NewGossip(protocol.TestNet), strconv.Itoa(port), processor, protocol.VoucherNode, challenger.SendChallenge)

	log.
		WithField("port", port).
		Info("Voucher seeder up & accepting connections")

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt

	return c.Close()
}
