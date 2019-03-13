package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"

	"gitlab.dusk.network/dusk-core/dusk-go/demo/consensus/ui"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/collection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
)

type registry struct {
	*ui.UI
}

func (r *registry) register(num int, s *Server) {
	node := &ui.Node{
		Online:    true,
		Port:      s.port,
		Phase:     "Block Generation",
		Step:      s.ctx.BlockStep,
		Round:     s.ctx.Round,
		Weight:    s.ctx.Weight,
		LastBlock: hex.EncodeToString(s.ctx.LastHeader.Hash),
	}

	r.UI.Nodes.Store(num, node)
}

func main() {
	r := &registry{
		UI: &ui.UI{
			Nodes:     &sync.Map{},
			LogChan:   make(chan string, 100),
			ChainChan: make(chan string, 100),
			BlockTime: make(chan int64, 1),
			BlockChan: make(chan *block.Block, 100),
		},
	}

	spawn(r)

	if err := ui.Run(r.UI); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func spawn(r *registry) {
	rand.Seed(time.Now().UTC().UnixNano())
	var peers []string
	port := 2000
	for i := 1; i < 9; i++ {
		go runNode(r, i, strconv.Itoa(port), peers)
		peers = append(peers, strconv.Itoa(port))
		r.UI.NodeStrings = append(r.UI.NodeStrings, "["+strconv.Itoa(i)+"] [127.0.0.1:"+strconv.Itoa(port)+"](fg-blue)")
		port++
		time.Sleep(250 * time.Millisecond)
	}
}

func runNode(r *registry, num int, port string, peers []string) {
	s := setupServer(7)
	s.port = port
	s.ctx = setupContext(s)

	// setup connection manager struct
	cmgr := setupConnMgr(port, peers, s)
	for _, peer := range peers {
		err := cmgr.Connect(peer)
		if err != nil {
			fmt.Println(err)
		}
	}

	var phase string

	// Register the node
	r.register(num, s)

	// Set up updating routine
	go func(r *registry, s *Server) {
		ticker := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				v, ok := r.UI.Nodes.Load(num)
				if !ok {
					os.Exit(1)
				}

				node := v.(*ui.Node)
				round := atomic.LoadUint64(&s.ctx.Round)
				node.Round = round
				node.LastBlock = hex.EncodeToString(s.ctx.LastHeader.Hash)[0:16] + "..."
				if strings.Contains(node.Phase, "Block") {
					node.Step = atomic.LoadUint32(&s.ctx.BlockStep)
				} else {
					node.Step = atomic.LoadUint32(&s.ctx.SigSetStep)
				}

				node.Phase = phase

				// Get out log messages
				for len(s.msgChan) > 0 {
					msg := <-s.msgChan
					r.LogChan <- msg
				}

				r.UI.Nodes.Store(num, node)
			}
		}
	}(r, s)

	// wait for a connection, so we can start off simultaneously with another node
	s.wg.Wait()

	r.UI.LastTime = time.Now().Unix()

	// Trigger consensus loop
	for {
		// Reset context
		s.ctx.Reset()

		// Block phase

		// Start block agreement concurrently
		c := make(chan bool, 1)
		go agreement.Block(s.ctx, c)

	block:
		for atomic.LoadUint32(&s.ctx.BlockStep) < user.MaxSteps {
			select {
			case v := <-c:
				// If it was false, something went wrong and we should quit
				if !v {
					os.Exit(1)
				}

				// If not, we proceed to the next phase
				break block
			default:
				if bytes.Equal(s.ctx.BlockHash, make([]byte, 32)) {
					phase = "Block Generation"
					if err := generation.Block(s.ctx); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}

					// Block collection
					phase = "Block Collection"
					if err := collection.Block(s.ctx); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}

				atomic.AddUint32(&s.ctx.BlockStep, 1)

				// Vote on received block. The context object should hold a winning
				// block hash after this function returns.
				phase = "Block Reduction/Agreement"
				if err := reduction.Block(s.ctx); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				if !bytes.Equal(s.ctx.WinningBlockHash, make([]byte, 32)) {
					break
				}
			}
		}

		// Empty QuitChan
		s.ctx.QuitChan = make(chan bool, 1)

		if bytes.Equal(s.ctx.WinningBlockHash, make([]byte, 32)) {
			continue
		}

		// Signature set phase
		s.ctx.Multiplier = 1

		// Fire off parallel set agreement phase
		c = make(chan bool, 1)
		go agreement.SignatureSet(s.ctx, c)

	sigset:
		for atomic.LoadUint32(&s.ctx.SigSetStep) < user.MaxSteps {
			select {
			case v := <-c:
				// If it was false, something went wrong and we should quit
				if !v {
					os.Exit(1)
				}

				// Propagate block
				// TODO: set signature
				// if bytes.Equal(s.ctx.BlockHash, s.ctx.CandidateBlock.Header.Hash) {
				// 	m := payload.NewMsgInv()
				// 	m.AddBlock(s.ctx.CandidateBlock)
				// 	if err := s.ctx.SendMessage(s.ctx.Magic, m); err != nil {
				// 		fmt.Println(err)
				// 		os.Exit(1)
				// 	}
				// }

				break sigset
			default:
				if bytes.Equal(s.ctx.WinningSigSetHash, make([]byte, 32)) {
					phase = "Signature Set Collection/Agreement"
					if err := generation.SignatureSet(s.ctx); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}

					if err := collection.SignatureSet(s.ctx); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}

				atomic.AddUint32(&s.ctx.SigSetStep, 1)

				// Vote on received signature set
				phase = "Signature Set Reduction/Agreement"
				if err := reduction.SignatureSet(s.ctx); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				if !bytes.Equal(s.ctx.SigSetHash, make([]byte, 32)) {
					break
				}
			}
		}

		// Empty QuitChan
		s.ctx.QuitChan = make(chan bool, 1)

		if bytes.Equal(s.ctx.WinningSigSetHash, make([]byte, 32)) {
			s.ctx.StopChan <- true
			continue
		}

		s.ctx.LastHeader.Height = s.ctx.Round
		s.ctx.LastHeader.Hash = s.ctx.WinningBlockHash
		s.ctx.Seed = s.ctx.CandidateBlock.Header.Seed
		s.ctx.LastHeader.Seed = s.ctx.CandidateBlock.Header.Seed
		s.ctx.BlockQueue.Map.Delete(s.ctx.Round)
		s.ctx.SigSetQueue.Map.Delete(s.ctx.Round)
		s.ctx.Multiplier = 1
		atomic.StoreUint32(&s.ctx.BlockStep, 1)
		atomic.StoreUint32(&s.ctx.SigSetStep, 1)
		r.UI.ChainChan <- fmt.Sprintf("%v - %s...\n", s.ctx.LastHeader.Height, hex.EncodeToString(s.ctx.WinningBlockHash)[0:16])
		r.UI.BlockChan <- s.ctx.CandidateBlock

		if num == 1 {
			r.UI.BlockTime <- time.Now().Unix()
		}

		s.ctx.Round++
	}
}

// gets the port and peers to connect to
// from the command line arguments
// func getPortPeers() (string, string, []string) {
// 	if len(os.Args) < 3 { // [programName, nodes,port,...]
// 		fmt.Println("Please enter more arguments")
// 		os.Exit(1)
// 	}

// 	args := os.Args[1:] // Remove program name leaving [nodes, port, peers...]

// 	nodes := args[0]
// 	port := args[1]

// 	var peers []string
// 	if len(args) > 2 {
// 		peers = args[2:]
// 	}

// 	return nodes, port, peers
// }

// setupServer creates the server struct
// populating it with the configurations for peer
func setupServer(nodes int) *Server {
	// Setup server
	s := &Server{
		peers:   make([]*peermgr.Peer, 0, 10),
		msgChan: make(chan string, 100),
	}

	s.wg.Add(nodes)

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
	ctx.Committee = append(ctx.Committee, keys.EdPubKeyBytes())
	ctx.Weight = uint64(weight)
	pkEd := hex.EncodeToString(keys.EdPubKeyBytes())
	pkBLS := hex.EncodeToString(keys.BLSPubKey.Marshal())
	ctx.NodeBLS[pkBLS] = *keys.EdPubKey
	ctx.NodeWeights[pkEd] = uint64(weight)

	// Make a staking transaction which mirrors this to other nodes
	stake := transactions.NewStake(1000, 100, keys.EdPubKeyBytes(),
		keys.BLSPubKey.Marshal())
	byte32, _ := crypto.RandEntropy(32)
	in := transactions.NewInput(byte32, byte32, 0, byte32)
	out := transactions.NewOutput(uint64(weight), byte32, byte32)
	stake.AddInput(in)
	stake.AddOutput(out)
	s.stake = transactions.NewTX(transactions.StakeType, stake)
	if err := s.stake.SetHash(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	m := zkproof.CalculateM(ctx.K)
	bid := transactions.NewBid(1000, m.Bytes(), 100)
	in2 := transactions.NewInput(byte32, byte32, 0, byte32)
	out2 := transactions.NewOutput(uint64(weight), byte32, byte32)
	bid.AddInput(in2)
	bid.AddOutput(out2)
	s.bid = transactions.NewTX(transactions.BidType, bid)
	if err := s.bid.SetHash(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Substitute SendMessage with our own function
	ctx.SendMessage = s.sendMessage
	return ctx
}
