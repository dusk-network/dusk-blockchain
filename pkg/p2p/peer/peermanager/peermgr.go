package peermanager

import (
	"errors"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
)

// NOTE: This package may be removed in the future
// and so full functionality is not yet implemented, see Issue #33 for more details.

// PeerMgr will act as a convenience Mgr
// It will be notified of added Peers
// It will take care of sending messages to the right peers. In this way, it acts as a load balancer
// If we send a getdata to one peer, it will be smart and send it to another peer who is not as busy
// Using subscription model, we can have the syncmanager/other modules notify the peermgr when they have received data
type PeerMgr struct {
	peers []*peer.Peer
}

// New will create a new peer manager
// As of now it just returns a peerMgr struct and so
// the New method is redundant. A config file will be passed as a parameter,
// if it is decided that we will use this.
func New() *PeerMgr {
	return &PeerMgr{}
}

// Disconnect will close the connection on a peer and
// remove it from the list
// TODO: remove from list once disconnected
func (pm *PeerMgr) Disconnect(p *peer.Peer) {
	p.Disconnect()
	// Once disconnected, we remove it from the list
	// and look for more peers to connect to
}

// RequestHeaders will request the headers from the most available peer
// As of now, it requests from the first peer in the list, TODO
func (pm *PeerMgr) RequestHeaders(hash []byte) (*peer.Peer, error) {

	if len(pm.peers) == 0 {
		return nil, errors.New("Peer manager currently has no peers")
	}

	return pm.peers[0], pm.peers[0].RequestHeaders(hash)
}

// RequestBlocks will request blocks from the most available peer
// As of now, it requests from the first peer in the list, TODO
func (pm *PeerMgr) RequestBlocks(hashes [][]byte) (*peer.Peer, error) {

	if len(pm.peers) == 0 {
		return nil, errors.New("Peer manager currently has no peers")
	}

	return pm.peers[0], pm.peers[0].RequestBlocks(hashes)
}

// AddPeer will add a new peer for the PeerManager to use
func (pm *PeerMgr) AddPeer(p *peer.Peer) error { //TODO: What error
	pm.peers = append(pm.peers, p)
	fmt.Printf("Adding peer %s into the peer manager.\n", p.RemoteAddr().String())
	return nil
}
