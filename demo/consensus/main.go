package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
)

func main() {

	rand.Seed(time.Now().UTC().UnixNano())

	if len(os.Args) < 2 { // [programName, port,...]
		fmt.Println("Please enter more arguments")
		os.Exit(1)
	}

	args := os.Args[1:] // [port, peers...]

	port := args[0]

	var peers []string
	if len(args) > 1 {
		peers = args[1:]
	}

	// Setup server
	s := Server{
		peers: make([]*peermgr.Peer, 0, 10),
		cfg:   setupPeerConfig(rand.Uint64()),
	}

	// setup connmgr
	cfg := CmgrConfig{
		Port:     port,
		OnAccept: s.OnAccept,
		OnConn:   s.OnConnection,
	}
	cmgr := newConnMgr(cfg)
	for _, peer := range peers {
		err := cmgr.Connect(peer)
		if err != nil {
			fmt.Println(err)
		}

	}

	consensus, err := consensus.New(setupConsensusConfig())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
	}
}
