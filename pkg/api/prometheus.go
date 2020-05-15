package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MsgProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_processed_msg_total",
		Help: "The total number of processed messages",
	})
)
