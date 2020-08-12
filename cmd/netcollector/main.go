package main

import (
	"os"
	"os/signal"

	"time"

	log "github.com/sirupsen/logrus"
)

var (
	jsonRPCAddr = ":1337"
)

func main() {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log.Infof("Starting netcollector with jsonrpc: %s", jsonRPCAddr)

	// in-memory dataset to process stats
	var d database

	// Run JsonRPC service
	go runJSONRPCServer(jsonRPCAddr, &d)

	// Run the monitoring
	go d.printTPS(3 * time.Second)

	<-interrupt
}
