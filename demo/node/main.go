package main

import (
	"fmt"
	"time"

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
	if len(ips) == 0 {
		fmt.Println("starting consensus")
		srv.StartConsensus(1)
	} else {
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
			srv.StartConsensus(highest.Header.Height + 2)
		} else {
			srv.StartConsensus(1)
		}
		fmt.Println("starting consensus")

	}

	for {
		time.Sleep(10 * time.Second)
	}
}
