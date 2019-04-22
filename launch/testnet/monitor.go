package main

import (
	ristretto "github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/monitor"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func ConnectToMonitor(bus wire.EventBroker, d ristretto.Scalar) {
	if cfg.Get().General.Network == "testnet" {
		monitorUrl := cfg.Get().Network.Monitor.Address
		log.Infof("Connecting to monitoring system")
		monitor.LaunchMonitor(bus, monitorUrl, d)
	}
}
