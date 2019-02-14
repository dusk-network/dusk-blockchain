package main

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"

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

	// Trigger consensus loop
	for {
		// Block generation
		if err := generation.Block(s.ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("our generated block is %s\n", hex.EncodeToString(s.ctx.BlockHash))

		// Block collection
		if err := collection.Block(s.ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("our best block is %s\n", hex.EncodeToString(s.ctx.BlockHash))

		// Block reduction
		if err := reduction.Block(s.ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("resulting hash from block reduction is %s\n",
			hex.EncodeToString(s.ctx.BlockHash))
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
	// Generate a random number between 100 and 1000 for the bid/stake weight
	weight := 100 + rand.Intn(900)

	// Create a context object to use
	keys, err := user.NewRandKeys()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, err := user.NewContext(0, uint64(weight), uint64(weight), 1,
		make([]byte, 32), protocol.TestNet, keys)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Set values for provisioning
	ctx.Weight = uint64(weight)
	pkEd := hex.EncodeToString([]byte(*keys.EdPubKey))
	pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
	ctx.NodeBLS[pkBLS] = *keys.EdPubKey
	ctx.NodeWeights[pkEd] = uint64(weight)

	// Make a staking transaction which mirrors this to other nodes
	stake := transactions.NewStake(1000, 100, []byte(*keys.EdPubKey),
		keys.BLSPubKey.Marshal())
	byte32, _ := crypto.RandEntropy(32)
	in := transactions.NewInput(byte32, byte32, 0, byte32)
	out := transactions.NewOutput(uint64(weight), byte32, byte32)
	stake.AddInput(in)
	stake.AddOutput(out)
	tx := transactions.NewTX(transactions.StakeType, stake)
	s.txs = append(s.txs, tx)

	// Substitute SendMessage with our own function
	ctx.SendMessage = s.sendMessage
	return ctx
}
