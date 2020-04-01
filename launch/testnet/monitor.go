package main

import (
	"strings"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/monitor"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "monitoring")

// New creates a new monitor.Supervisor which notifies a remote monitoring server with alerts and data
func New(bus eventbus.Broker) (monitor.Supervisor, error) {
	if cfg.Get().General.Network == "testnet" && cfg.Get().Logger.Monitor.Enabled {
		mon := cfg.Get().Logger.Monitor
		rpcType := strings.ToLower(mon.Rpc)
		transport := strings.ToLower(mon.Transport)
		addr := strings.ToLower(mon.Address)

		monitorURL := transport + "://" + addr
		lg.WithField("process", "monitoring").Info("Monitor configuration parsed: sending data to", monitorURL)

		blockTimeThreshold := 20 * time.Second

		supervisor, err := monitor.New(bus, blockTimeThreshold, rpcType, monitorURL)
		if err != nil {
			lg.WithError(err).Errorln("Monitoring could not get started")
			return nil, err
		}

		if cfg.Get().Logger.Monitor.StreamErrors {
			lg.Infoln("sending errors to", monitorURL)
			log.AddHook(supervisor)
		}

		lg.Debugln("monitor instantiated")
		return supervisor, nil
	}

	return nil, nil
}
