package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
)

var configToml = flag.String("config", "", "dusk.toml")

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	flag.Parse()

	d := Deployer{ConfigPath: *configToml}

	go func() {
		// Notify main loop
		s := <-interrupt
		d.Signal(s)
	}()

	d.Run()
}
