package main

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/monitor"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

var lg = logrus.WithField("process", "monitoring")

// StopFunc is invoked when the monitor should be stopped. It is intended to
// perform the closing operations on the GRPC server and connections
type StopFunc func()

// LaunchMonitor creates a Supervisor if the configuration demands it, and starts it
func LaunchMonitor(bus eventbus.Broker) (StopFunc, error) {
	supervisor, err := NewSupervisor(bus)
	if err != nil {
		return func() {}, err
	}

	if supervisor == nil {
		return func() {}, nil
	}

	if err := supervisor.Start(); err != nil {
		return func() {}, err
	}

	return func() {
		_ = supervisor.Stop()
	}, nil
}

// ParseMonitorConfiguration collects parameters of the monitoring client
func ParseMonitorConfiguration() (rpcType, target string, streamErrors bool) {
	mon := cfg.Get().Logger.Monitor
	rpcType = strings.ToLower(mon.Rpc)
	transport := strings.ToLower(mon.Transport)
	addr := strings.ToLower(mon.Address)

	target = transport + "://" + addr
	lg.WithField("process", "monitoring").Info("Monitor configuration parsed: sending data to", target)

	streamErrors = cfg.Get().Logger.Monitor.StreamErrors
	return
}

// NewSupervisor creates a new monitor.Supervisor which notifies a remote monitoring server with alerts and data
func NewSupervisor(bus eventbus.Broker) (monitor.Supervisor, error) {
	if cfg.Get().General.Network == "testnet" && cfg.Get().Logger.Monitor.Enabled {
		rpc, target, streamErrors := ParseMonitorConfiguration()

		blockTimeThreshold := 20 * time.Second
		supervisor, err := monitor.New(bus, blockTimeThreshold, rpc, target)
		if err != nil {
			lg.WithError(err).Errorln("Monitoring could not get started")
			return nil, err
		}

		if streamErrors {
			lg.Infoln("adding supervisor as logrus hook so to send errors to", target)
			logrus.AddHook(supervisor)
		}

		lg.Debugln("monitor instantiated")
		return supervisor, nil
	}

	return nil, nil
}
