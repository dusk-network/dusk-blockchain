package main

import (
	"strings"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/monitor"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	l "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "monitoring")

// LaunchMonitor creates a Supervisor if the configuration demands it, and starts it
func LaunchMonitor(bus eventbus.Broker) error {
	supervisor, err := NewSupervisor(bus)
	if err != nil {
		return err
	}

	if supervisor == nil {
		return nil
	}

	if err := supervisor.Start(); err != nil {
		return err
	}

	return nil
}

// NewSupervisor creates a new monitor.Supervisor which notifies a remote monitoring server with alerts and data
func NewSupervisor(bus eventbus.Broker) (monitor.Supervisor, error) {
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
			lg.Infoln("adding supervisor as logrus hook so to send errors to", monitorURL)
			l.AddHook(supervisor)
		}

		lg.Debugln("monitor instantiated")
		return supervisor, nil
	}

	return nil, nil
}
