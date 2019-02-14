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

	port, peers := getPortPeers()

	s := setupServer()

	s.ctx = setupContext(s)

	// setup connection manager struct
	cmgr := setupConnMgr(port, peers, s)
	for _, peer := range peers {
		err := cmgr.Connect(peer)
		if err != nil {
			fmt.Println(err)
		}
	}

	// wait for a connection, so we can start off simultaneously
	<-s.connectChan

	// Trigger block generation and block collection loop
	for {
		generation.Block(s.ctx)
		fmt.Printf("our generated block is %s\n", hex.EncodeToString(s.ctx.BlockHash))
		collection.Block(s.ctx)
		fmt.Printf("our best block is %s\n", hex.EncodeToString(s.ctx.BlockHash))
	}
}

// gets the port and peers to connect to
// from the command line arguments
func getPortPeers() (string, []string) {
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

	return port, peers
}

// setupServer creates the server struct
// populating it with the configurations for peer
func setupServer() *Server {
	// Setup server
	s := &Server{
		peers:       make([]*peermgr.Peer, 0, 10),
		connectChan: make(chan bool, 1),
	}

	s.cfg = setupPeerConfig(s, rand.Uint64())
	return s

}

func setupConnMgr(port string, peers []string, s *Server) *connmgr {
	// setup connmgr
	cfg := CmgrConfig{
		Port:     port,
		OnAccept: s.OnAccept,
		OnConn:   s.OnConnection,
	}
	cmgr := newConnMgr(cfg)
	return cmgr
}

func setupContext(s *Server) *user.Context {
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

	// Substitute SendMessage with our own function
	ctx.SendMessage = s.sendMessage
	return ctx
}
