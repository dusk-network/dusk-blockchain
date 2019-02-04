package syncmgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
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

	// GetGoodAddresses requests for the best addresses
	GetGoodAddresses func() []payload.NetAddress

	// AddAddrs adds new addresses to the Address Manager
	AddAddrs func([]*payload.NetAddress)

	// ValidateHeaders validates headers that were received through the wire
	ValidateHeaders func(*payload.MsgHeaders) error

	// AddHeaders adds block headers to the chain
	AddHeaders func(*payload.MsgHeaders) error

	// GetBlock requests for the block with this hash
	GetBlock func([]byte) (*payload.Block, error)

	// AcceptBlock verifies a block received from the network
	AcceptBlock func(*payload.Block) error

	// GetHeaders request for block headers from the database starting and stopping at the provided hashes
	GetHeaders func([]byte, []byte) ([]*payload.BlockHeader, error)
}
