package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
)

func initLog(file *os.File) {

	// apply logger level from configurations
	level, err := log.ParseLevel(cfg.Get().Logger.Level)
	if err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(log.TraceLevel)
		log.Warnf("Parse logger level from config err: %v", err)
	}

	if file != nil {
		os.Stdout = file
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}
}

func main() {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Loading all node configurations. Fail-fast if critical error occurs
	if err := cfg.Load(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	profile, err := newProfile()

	if err != nil {
		fmt.Printf(" %v\n", err)
		os.Exit(1)
	}

	defer profile.close()

	port := cfg.Get().Network.Port
	rand.Seed(time.Now().UnixNano())

	// Set up logging
	output := cfg.Get().Logger.Output
	if cfg.Get().Logger.Output != "stdout" {
		file, err := os.Create(output + port + ".log")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		initLog(file)
	} else {
		initLog(nil)
	}

	log.Infof("Loaded config file %s", cfg.Get().UsedConfigFile)
	log.Infof("Selected network  %s", cfg.Get().General.Network)

	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup()
	defer srv.Close()

	// listening to the blindbid and the stake channels
	go srv.Listen()
	// fetch neighbours addresses from the Seeder
	ips := ConnectToSeeder()
	//start the connection manager
	connMgr := NewConnMgr(CmgrConfig{
		Port:     port,
		OnAccept: srv.OnAccept,
		OnConn:   srv.OnConnection,
	})

	// wait a bit for everyone to start their cmgr
	time.Sleep(time.Second * 1)

	round := joinConsensus(connMgr, srv, ips)
	srv.StartConsensus(round)

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt

	log.WithField("prefix", "main").Info("Terminated")
}

func joinConsensus(connMgr *connmgr, srv *Server, ips []string) uint64 {
	// if we are the first, initialize consensus on round 1
	if len(ips) == 0 {
		log.WithField("Process", "main").Infoln("Starting consensus from scratch")
		return uint64(1)
	}

	// trying to connect to the peers
	for _, ip := range ips {
		if err := connMgr.Connect(ip); err != nil {
			log.WithField("IP", ip).Warnln(err)
		}
	}

	// if height is not 0, init consensus on 2 rounds after it
	// +1 because the round is always height + 1
	// +1 because we dont want to get stuck on a round thats currently happening
	if srv.chain.PrevBlock.Header.Height != 0 {
		round := srv.chain.PrevBlock.Header.Height + 2
		log.WithField("prefix", "main").Infof("Starting consensus from round %d\n", round)
		return round
	}

	log.WithField("prefix", "main").Infoln("Starting consensus from scratch")
	return uint64(1)
}
