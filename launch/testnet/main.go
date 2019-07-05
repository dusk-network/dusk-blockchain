package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
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

	port := cfg.Get().Network.Port
	rand.Seed(time.Now().UnixNano())

	// Set up logging.
	// Any subsystem should be initialized after config and logger loading
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

	// TODO: this should be adjusted before testnet release, it is simply a way
	// to bootstrap the network in an unsophisticated manner
	if strings.Contains(ips[0], "noip") {
		log.WithField("Process", "main").Infoln("Starting consensus from scratch")
		// Create mock block on height 1 with our stake and bid
		blk := mockBlockOne(srv.MyBid, srv.MyStake)
		buf := new(bytes.Buffer)
		if err := blk.Encode(buf); err != nil {
			panic(err)
		}

		srv.eventBus.Publish(string(topics.Block), buf)
		srv.StartConsensus(2)
	} else {
		// Propagate bid and stake out to the network
		srv.sendStake()
		srv.sendBid()
		// wait for stake to appear in an incoming accepted block
		height := waitForStake(srv.eventBus, srv.MyStake)
		srv.StartConsensus(height + 1)
	}

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt

	log.WithField("prefix", "main").Info("Terminated")
}
