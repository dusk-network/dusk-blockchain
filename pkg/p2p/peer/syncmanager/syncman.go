// The syncmanager will use a modified verison of the initial block download in bitcoin
// Seen here: https://en.bitcoinwiki.org/wiki/Bitcoin_Core_0.11_(ch_5):_Initial_Block_Download
// MovingWindow is a desired featured from the original codebase

package syncmanager

import (
	"encoding/hex"
	"fmt"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/blockchain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermanager"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

var (
	// This is the maximum amount of inflight objects that we would like to have
	// Number taken from original codebase
	maxBlockRequest = 1024

	// This is the maximum amount of blocks that we will ask for from a single peer
	// Number taken from original codebase
	maxBlockRequestPerPeer = 16
)

type Syncmanager struct {
	pmgr              *peermanager.PeerMgr
	Mode              int // 1 = headersFirst, 2 = Blocks, 3 = Maintain
	chain             *blockchain.Chain
	headers           [][]byte
	inflightBlockReqs map[string]*peer.Peer // When we send a req for block, we will put hash in here, along with peer who we requested it from
}

// New will setup the syncmanager with the required parameters
func New(cfg Config) *Syncmanager {
	return &Syncmanager{
		peermanager.New(),
		1,
		cfg.Chain,
		[][]byte{},
		make(map[string]*peer.Peer, 2000),
	}
}

func (s *Syncmanager) AddPeer(peer *peer.Peer) error {
	return s.pmgr.AddPeer(peer)
}

func (s *Syncmanager) OnGetHeaders(p *peer.Peer, msg *payload.MsgGetHeaders) {
	fmt.Println("Sync manager OnGetHeaders called")
	// The caller peer wants some headers from our blockchain.
	msgHeaders, err := getHeaders(*s.chain, msg)
	if err == nil {
		if len(msgHeaders.Headers) > 0 {
			err = p.Write(msgHeaders)
		} else {
			//TODO: Needs refactoring. MsgNotFound is for InvTypes, not for Headers.
			// We could just send an empty 'headers' msg back.
			hashes := [][]byte{msg.Locator, msg.HashStop}
			sendMsgNotFound(p, payload.InvBlock, hashes)
		}
	}
	if err != nil {
		fmt.Printf("Could not send '%s' to requesting peer %s: %s.", commands.Headers, p.RemoteAddr().String(), err)
	}
}

func (s *Syncmanager) OnHeaders(p *peer.Peer, msg *payload.MsgHeaders) {
	fmt.Println("Sync manager OnHeaders called")
	// On receipt of Headers check what mode we are in
	// HeadersMode, we check if there is 2k. If so call again. If not then change mode into BlocksOnly
	if s.Mode == 1 {
		err := s.HeadersFirstMode(p, msg)
		if err != nil {
			fmt.Println("Error reading block headers:", err)
			return // We should custom name error so, that we can do something on WrongHash Error, Peer disconnect error
		}
		return
	}
}

func (s *Syncmanager) HeadersFirstMode(p *peer.Peer, msg *payload.MsgHeaders) error {
	fmt.Println("Headers first mode")

	// Validate Headers
	err := s.chain.ValidateHeaders(msg)

	if err != nil {
		// Re-request headers from a different peer
		s.pmgr.Disconnect(p)
		fmt.Println("Error validating headers:", err)
		return err
	}

	// Add Headers into db
	err = s.chain.AddHeaders(msg)
	if err != nil {
		// Try addding them into the db again?
		// Since this is simply a db insert, any problems here means trouble
		//TODO: Should we Switch off system or warn the user that the system is corrupted?
		fmt.Println("Error Adding headers", err)

		//TODO: Batching is not yet implemented,
		// So here we would need to remove headers which have been added
		// from the slice
		return err
	}

	// Add header hashes into slice
	// Request first batch of blocks here
	hashes := make([][]byte, len(msg.Headers))
	for _, header := range msg.Headers {
		hashes = append(hashes, header.Hash)
	}
	s.headers = append(s.headers, hashes...)

	if len(msg.Headers) == 2*1e3 { // should be less than 2000, leave it as this for tests
		fmt.Println("Switching to BlocksOnly Mode")
		s.Mode = 2 // switch to BlocksOnly. XXX: because HeadersFirst is not in parallel, no race condition here.
		return s.RequestMoreBlocks()
	}
	latestHeader := msg.Headers[len(msg.Headers)-1]
	_, err = s.pmgr.RequestHeaders(latestHeader.Hash)
	return err
}

func (s *Syncmanager) RequestMoreBlocks() error {
	var blockReq [][]byte
	var reqAmount int

	if len(s.headers) >= maxBlockRequestPerPeer {
		reqAmount = maxBlockRequestPerPeer
		blockReq = s.headers[:reqAmount]
	} else {
		reqAmount = len(s.headers)
		blockReq = s.headers[:reqAmount]
	}
	peer, err := s.pmgr.RequestBlocks(blockReq)
	if err != nil { // This could happen if the peermanager has no valid peers to connect to. We should wait a bit and re-request
		return err // alternatively we could make RequestBlocks blocking, then make sure it is not triggered when a block is received
	}

	//XXX: Possible race condition, between us requesting the block and adding it to
	// the inflight block map? Give that node a medal.

	for _, hash := range s.headers {
		hashKey := hex.EncodeToString(hash)
		s.inflightBlockReqs[hashKey] = peer
	}
	s.headers = s.headers[reqAmount:]
	// NONONO: Here we do not pass all of the hashes to peermanager because
	// it is not the peermanagers responsibility to mange inflight blocks
	return err
}

// OnBlock receives a block from a peer, then passes it to the blockchain to process.
// For now we will only use this simple setup, to allow us to test the other parts of the system.
// See Issue #24
func (s *Syncmanager) OnBlock(p *peer.Peer, msg *payload.MsgBlock) {
	err := s.chain.AddBlock(msg)
	if err != nil {
		// Put headers back in front of queue to fetch block for.
		fmt.Println("Block had an error", err)
	}
}

// OnBlock receives a block from a peer, then passes it to the blockchain to process.
// For now we will only use this simple setup, to allow us to test the other parts of the system.
// See Issue #24
func (s *Syncmanager) OnMemPool(p *peer.Peer, msg *payload.MsgMemPool) {
	//err := s.chain.AddMempool(msg)
	//if err != nil {
	//	// Put headers back in front of queue to fetch block for.
	//	fmt.Println("Block had an error", err)
	//}
}

func sendMsgNotFound(p *peer.Peer, invType payload.InvType, hashes [][]byte) {
	msg := payload.NewMsgNotFound()
	var vectors []payload.InvVect
	for _, hash := range hashes {
		invVect := payload.InvVect{invType, hash}
		vectors = append(vectors, invVect)
	}
	msg.Vectors = vectors
	err := p.Write(msg)
	if err != nil {
		fmt.Printf("Could not send '%s' to requesting peer %s: %s.", commands.NotFound, p.RemoteAddr().String(), err)
	}
}
