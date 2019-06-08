package main

import (
	ristretto "github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/monitor"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func ConnectToLogMonitor(bus wire.EventBroker, d ristretto.Scalar) (monitor.Supervisor, error) {
	if cfg.Get().General.Network == "testnet" && cfg.Get().Logger.Monitor.Enabled {
		monitorUrl := cfg.Get().Logger.Monitor.Target
		log.Infof("Connecting to log reserved monitoring file on %v\n", monitorUrl)
		return monitor.LaunchLogMonitor(bus, monitorUrl)
	}
}
