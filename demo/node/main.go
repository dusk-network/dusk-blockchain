package main

import (
	"fmt"
	"os"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

func main() {
	srv := Setup()
	go srv.Listen()
	ips := ConnectToSeeder()
	connMgr := NewConnMgr(CmgrConfig{
		Port:     "8081",
		OnAccept: srv.OnAccept,
		OnConn:   srv.OnConnection,
	})

	// if we are the first, initialize consensus on round 1
	if len(ips) < 2 {
		fmt.Println("starting consensus")
		srv.StartConsensus(1)
	} else {
		for _, ip := range ips {
			if err := connMgr.Connect(ip); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

		}

		// get highest block, and init consensus on 2 rounds after it
		// +1 because the round is always height + 1
		// +1 because we dont want to get stuck on a round thats currently happening
		var highest block.Block
		for _, block := range srv.Blocks {
			if block.Header.Height > highest.Header.Height {
				highest = block
			}
		}

		fmt.Println("starting consensus")
		srv.StartConsensus(highest.Header.Height + 2)
	}

	for {

	}
}
