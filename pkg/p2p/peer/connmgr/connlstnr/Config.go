package connlstnr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"net"
)

// Config contains all functions which will be set by the caller to setup the connection listener
type Config struct {

	// CreatePeer is called after a connection to a peer was successful
	CreatePeer func(net.Conn, bool) *peermgr.Peer

	// RequestAddreses request addresses from an other peer
	RequestAddresses func() error

	// ConnectionComplete will be called after a successful handshake
	// It is to tell the AddrMgr that we have connected to a peer.
	ConnectionComplete func(string, bool)

	// NewAddr will return an address for the caller to connect to
	NewAddr func() (string, error)

	// Failed is used to tell the AddrMgr that they had tried connecting an address and have failed
	Failed func(string)

	// GetLatestHeader returns the most recent header
	GetLatestHeader func() (*payload.BlockHeader, error)
}
