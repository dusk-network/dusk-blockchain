package main

import (
	"fmt"
	"os"
	"os/signal"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

func main() {
	// Gracefully exiting on SIGNAL requires a channel
	interrupt := make(chan os.Signal, 1)
	//hooking the signal channel up to the interrupt coming from the OS
	signal.Notify(interrupt, os.Interrupt)

	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup()
	// listening to the blindbid and the stake channels
	go srv.Listen()
	// fetch neighbours addresses from the Seeder
	ips := ConnectToSeeder()
	//start the connection manager
	connMgr := NewConnMgr(CmgrConfig{
		Port:     "8081",
		OnAccept: srv.OnAccept,
		OnConn:   srv.OnConnection,
	})

	round := joinConsensus(connMgr, srv, ips)
	srv.StartConsensus(round)

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt
}

func joinConsensus(connMgr *connmgr, srv *Server, ips []string) uint64 {

	// if we are the first, initialize consensus on round 1
	if len(ips) == 0 {
		fmt.Println("Starting consensus from scratch")
		return uint64(1)
	}

	// trying to connect to the peers
	for _, ip := range ips {
		if err := connMgr.Connect(ip); err != nil {
			fmt.Println(err)
		}
	}

	// get highest block
	var highest *block.Block
	for _, block := range srv.Blocks {
		if block.Header.Height > highest.Header.Height {
			highest = &block
		}
	}

	// if height is not 0, init consensus on 2 rounds after it
	// +1 because the round is always height + 1
	// +1 because we dont want to get stuck on a round thats currently happening
	if highest != nil && highest.Header.Height != 0 {
		round := highest.Header.Height + 2
		fmt.Printf("Starting consensus from %d\n", round)
		return round
	}

	fmt.Println("Starting consensus from scratch")
	return uint64(1)
}
