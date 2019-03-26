package monitor

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	datadog "gopkg.in/zorkian/go-datadog-api.v2"
)

func launchBlockTimeMonitor(eventBus *wire.EventBus, client *datadog.Client, roundChan <-chan uint64) {
	var lastBlockTime *time.Time
	for {
		<-roundChan
		nowRef := time.Now()
		now := &nowRef
		if lastBlockTime == nil {
			lastBlockTime = now
			continue
		}

		blockDuration := now.Sub(*lastBlockTime)
		lastBlockTime = now

		dataType := "consensus_time"
		metric := newData(dataType, float64(now.Unix()), float64(blockDuration.Seconds()))

		//TODO: handler error
		_ = client.PostMetrics(metric)
	}
}

func newData(dataType string, timestamp float64, scalar float64) []datadog.Metric {
	dataPoint := datadog.DataPoint{&timestamp, &scalar}
	return []datadog.Metric{
		datadog.Metric{
			Metric: &dataType,
			Points: []datadog.DataPoint{dataPoint},
		},
	}
}

func LaunchDataDog(eventBus *wire.EventBus) {
	client := datadog.NewClient("6378ad8a96906938795b580341e6cf88", "4361397144bbd1ee3b27fcde599e940f736fa101")

	roundChan := consensus.InitRoundUpdate(eventBus)
	launchBlockTimeMonitor(eventBus, client, roundChan)
}
