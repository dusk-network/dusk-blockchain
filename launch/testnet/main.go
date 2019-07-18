package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/cli"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/logging"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Fprintln(os.Stdout, "initializing node...")
	// Loading all node configurations. Fail-fast if critical error occurs
	err := cfg.Load()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	port := cfg.Get().Network.Port
	rand.Seed(time.Now().UnixNano())

	// Set up logging.
	// Any subsystem should be initialized after config and logger loading
	output := cfg.Get().Logger.Output
	var logFile *os.File
	if cfg.Get().Logger.Output != "stdout" {
		logFile, err = os.Create(output + port + ".log")
		if err != nil {
			panic(err)
		}
		defer logFile.Close()
	} else {
		logFile = os.Stdout
	}

	logging.InitLog(logFile)

	log.Infof("Loaded config file %s", cfg.Get().UsedConfigFile)
	log.Infof("Selected network  %s", cfg.Get().General.Network)

	// Set up profiling tools.
	profile, err := newProfile()
	if err != nil {
		// Assume here if tools are enabled but they fail on loading then it's better
		// to fix the error or just disable them.
		log.Errorf("Profiling tools error: %s", err.Error())
		return
	}
	defer profile.close()

	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup()
	defer srv.Close()

	//start the connection manager
	connMgr := NewConnMgr(CmgrConfig{
		Port:     port,
		OnAccept: srv.OnAccept,
		OnConn:   srv.OnConnection,
	})

	// fetch neighbours addresses from the Seeder
	ips := ConnectToSeeder()

	// trying to connect to the peers
	for _, ip := range ips {
		if err := connMgr.Connect(ip); err != nil {
			log.WithField("IP", ip).Warnln(err)
		}
	}

	fmt.Fprintln(os.Stdout, "initialization complete. opening console...")

	// Start interactive shell
	if cfg.Get().Logger.Output != "stdout" {
		go cli.Start(srv.eventBus, srv.rpcBus, logFile)
	}

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt

	// Graceful shutdown of listening components
	srv.eventBus.Publish(msg.QuitTopic, new(bytes.Buffer))

	log.WithField("prefix", "main").Info("Terminated")
}
