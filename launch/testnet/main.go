package main

import (
	"flag"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// Flags
var voucher = flag.String("voucher", "voucher.dusk.network:8081", "hostname for the voucher seeder")
var port = flag.String("port", "7000", "port for the node to bind on")
var logToFile = flag.Bool("logtofile", false, "specifies if the log should be written to a file")

func initLog(file *os.File) {
	log.SetLevel(log.TraceLevel)
	if file != nil {
		os.Stdout = file
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	// Set up logging
	if *logToFile {
		file, err := os.Create("node" + *port + ".log")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		initLog(file)
	} else {
		initLog(nil)
	}

	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup("demo" + *port)
	// listening to the blindbid and the stake channels
	go srv.Listen()
	// fetch neighbours addresses from the Seeder
	ips := ConnectToSeeder()
	//start the connection manager
	connMgr := NewConnMgr(CmgrConfig{
		Port:     *port,
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
	select {}
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
