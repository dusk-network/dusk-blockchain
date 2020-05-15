package api

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func StartAPI() {

	// define API handlers
	http.Handle("/metrics", promhttp.Handler())

	// start API endpoint
	err := http.ListenAndServe(cfg.Get().API.Address, nil)
	if err != nil {
		log.WithField("err", err).Error("Could not start Prometheus monitoring...")
		return
	}

}
