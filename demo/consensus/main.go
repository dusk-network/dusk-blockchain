package main

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/collection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

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
	s := &Server{
		peers:       make([]*peermgr.Peer, 0, 10),
		connectChan: make(chan bool, 1),
	}

	s.cfg = setupPeerConfig(s, rand.Uint64())

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

	// Generate a random number between 100 and 1000 for the bid weight
	bidWeight := 100 + rand.Intn(900)

	// Create a context object to use
	keys, err := user.NewRandKeys()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, err := user.NewContext(0, uint64(bidWeight), 0, 1, make([]byte, 32), protocol.TestNet, keys)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.ctx = ctx

	// Substitute SendMessage with our own function
	ctx.SendMessage = s.sendMessage

	// wait for a connection, so we can start off simultaneously
	<-s.connectChan

	// Trigger block generation and block collection loop
	for {
		generation.Block(ctx)
		fmt.Printf("our generated block is %s\n", hex.EncodeToString(ctx.BlockHash))
		collection.Block(ctx)
		fmt.Printf("our best block is %s\n", hex.EncodeToString(ctx.BlockHash))
	}
}
