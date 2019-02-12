package main

import (
	"fmt"
	"os"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
)

func main() {

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

	for {
	}
}

func MockConsensusMsg() *payload.MsgConsensus {
	byte32, err := crypto.RandEntropy(32)
	byte33, err := crypto.RandEntropy(33)
	if err != nil {
		fmt.Println("Could not create a Mock Consensus message")
		os.Exit(1)
	}
	byte64, err := crypto.RandEntropy(64)
	if err != nil {
		fmt.Println("Could not create a Mock Consensus message")
		os.Exit(1)
	}
	reductionPayload, err := consensusmsg.NewReduction(byte33, byte32, byte32)
	if err != nil {
		fmt.Println("Could not create a Mock Consensus message")
		os.Exit(1)
	}
	msg, err := payload.NewMsgConsensus(10, 10, byte32, 10, byte64, byte32, reductionPayload)
	if err != nil {
		fmt.Println("Could not create a Mock Consensus message")
		os.Exit(1)
	}
	return msg
}
