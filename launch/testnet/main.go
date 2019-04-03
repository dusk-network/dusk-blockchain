package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

// Flags
var voucher = flag.String("voucher", "voucher.dusk.network", "hostname for the voucher seeder")
var port = flag.String("port", "8081", "port for the node to bind on")

func initLog() {
	log.SetOutput(os.Stdout)
	// log.SetLevel()
}

func main() {
	flag.Parse()
	initLog()
	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup()
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
		log.WithField("Process", "main").Infoln("tarting consensus from scratch")
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
