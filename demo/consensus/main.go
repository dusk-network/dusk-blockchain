package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
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
		// Reset context
		s.ctx.Reset()

		// Block phase

		// Block generation
		prErr := generation.Block(s.ctx)
		if prErr != nil && prErr.Priority == prerror.High {
			fmt.Println(prErr.Err)
			os.Exit(1)
		}

		fmt.Printf("our generated block is %s\n", hex.EncodeToString(s.ctx.BlockHash))

		// Block collection
		if err := collection.Block(s.ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("our best block is %s\n", hex.EncodeToString(s.ctx.BlockHash))

		// Start block agreement concurrently
		c := make(chan bool, 1)
		go agreement.Block(s.ctx, c)

		for s.ctx.Step < user.MaxSteps {
			select {
			case v := <-c:
				// If it was false, something went wrong and we should quit
				if !v {
					fmt.Println("error encountered during block agreement")
					os.Exit(1)
				}

				// If not, we proceed to the next phase by maxing out the
				// step counter.
				s.ctx.Step = user.MaxSteps
			default:
				// Vote on received block. The context object should hold a winning
				// block hash after this function returns.
				if err := reduction.Block(s.ctx); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				fmt.Printf("resulting hash from block reduction is %s\n",
					hex.EncodeToString(s.ctx.BlockHash))

				if s.ctx.BlockHash != nil {
					continue
				}

				// If we did not get a result, increase the multiplier and
				// exit the loop.
				s.ctx.Step = user.MaxSteps
				if s.ctx.Multiplier < 10 {
					s.ctx.Multiplier = s.ctx.Multiplier * 2
				}
			}
		}

		if s.ctx.BlockHash == nil {
			fmt.Println("exited without a block hash")
			os.Exit(1)
		}

		fmt.Printf("resulting hash from block agreement is %s\n",
			hex.EncodeToString(s.ctx.BlockHash))

		// Signature set phase

		// Reset step counter
		s.ctx.Step = 1

		// Fire off parallel set agreement phase
		go agreement.SignatureSet(s.ctx, c)

		for s.ctx.Step < user.MaxSteps {
			select {
			case v := <-c:
				// If it was false, something went wrong and we should quit
				if !v {
					fmt.Println("error encountered during signature set agreement")
					os.Exit(1)
				}

				// Propagate block
				// TODO: set signature
				if bytes.Equal(s.ctx.BlockHash, s.ctx.CandidateBlock.Header.Hash) {
					m := payload.NewMsgInv()
					m.AddBlock(s.ctx.CandidateBlock)
					if err := s.ctx.SendMessage(s.ctx.Magic, m); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}

				s.ctx.Step = user.MaxSteps
			default:
				if s.ctx.SigSetHash == nil {
					if err := generation.SignatureSet(s.ctx); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}

					if err := collection.SignatureSet(s.ctx); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}

					fmt.Printf("collected signature set hash is %s\n",
						hex.EncodeToString(s.ctx.SigSetHash))
				}

				// Vote on received signature set
				if err := reduction.SignatureSet(s.ctx); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				fmt.Printf("resulting hash from signature set reduction is %s\n",
					hex.EncodeToString(s.ctx.SigSetHash))

				if s.ctx.SigSetHash != nil {
					continue
				}

				// Increase multiplier
				if s.ctx.Multiplier < 10 {
					s.ctx.Multiplier = s.ctx.Multiplier * 2
				}
			}
		}

		fmt.Printf("final results:\n\tblock hash: %s\n\tsignature set hash: %s\n",
			hex.EncodeToString(s.ctx.BlockHash), hex.EncodeToString(s.ctx.SigSetHash))
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

	// Set context last header to a common value to avoid processing errors
	ctx.LastHeader = &block.Header{
		Version:   0,
		Timestamp: 150000000,
		Height:    0,
		PrevBlock: make([]byte, 32),
		Seed:      make([]byte, 32),
		TxRoot:    make([]byte, 32),
		CertHash:  make([]byte, 32),
		Hash:      make([]byte, 32),
	}

	// Set values for provisioning
	ctx.Committee = append(ctx.Committee, []byte(*keys.EdPubKey))
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
	if err := tx.SetHash(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.txs = append(s.txs, tx)

	// Substitute SendMessage with our own function
	ctx.SendMessage = s.sendMessage
	return ctx
}
