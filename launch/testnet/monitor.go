package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/monitor"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

var lg = log.WithField("process", "monitoring")

// ConnectToLogMonitor launches the monitoring process in a goroutine. The goroutine performs 5 attempts before giving up
func ConnectToLogMonitor(bus wire.EventBroker) error {
	if cfg.Get().General.Network == "testnet" && cfg.Get().Logger.Monitor.Enabled {
		monitorURL := cfg.Get().Logger.Monitor.Target
		lg.WithField("process", "monitoring").Infof("Connecting to log process on %v\n", monitorURL)
		go startMonitoring(bus, monitorURL)
	}

	return nil
}

func startMonitoring(bus wire.EventBroker, monURL string) {
	for i := 0; i < 5; i++ {
		lg.Traceln("Trying to (re)start the monitoring process")
		supervisor, err := monitor.Launch(bus, monURL)
		if err != nil {
			lg.Warnln(fmt.Sprintf("error in starting the monitoring. Attempt: %d. Error: %s", i, err.Error()))
			delay := 2 + 2*i
			lg.Warnln(fmt.Sprintf("waiting for %d before retrying", delay))
			time.Sleep(time.Duration(delay) * time.Second)
			continue
		}
		if cfg.Get().Logger.Monitor.StreamErrors {
			log.AddHook(supervisor)
		}
		return
	}
	lg.Errorln("Monitoring could not get started")
}
