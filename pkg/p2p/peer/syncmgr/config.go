package syncmgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
)

// Config contains all functions which will be set by the caller to setup the Synchronisation Manager
type Config struct {

	// AddPeer to add a new peer to the Address Manager
	AddPeer func(p *peermgr.Peer)

	// Disconnect closes the peer's connection and remove it from the list
	Disconnect func(p *peermgr.Peer)

	// RequestHeaders requests headers from the most available peer
	RequestHeaders func([]byte) (*peermgr.Peer, error)

	// RequestBlocks requests the Peer Manager for blocks from the most available peer
	RequestBlocks func([][]byte) (*peermgr.Peer, error)

	// RequestAddresses requests for the addresses from the most available peer
	RequestAddresses func() error
}
