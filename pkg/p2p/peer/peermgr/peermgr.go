package peermgr

import (
	log "github.com/sirupsen/logrus"
)

// NOTE: This package may be removed in the future
// and so full functionality is not yet implemented, see Issue #33 for more details.

// PeerMgr will act as a convenience manager
// It will be notified of added Peers
// It will take care of sending messages to the right peers. In this way, it acts as a load balancer
// If we send a getdata to one peer, it will be smart and send it to another peer who is not as busy
// Using subscription model, we can have the sync manager/other modules notify the peer manager when they have received data
type PeerMgr struct {
	peers []*Peer
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
func (pm *PeerMgr) Disconnect(p *Peer) {
	p.Disconnect()
	// Once disconnected, we remove it from the list
	// and look for more peers to connect to
}

// AddPeer will add a new peer for the PeerManager to use
func (pm *PeerMgr) AddPeer(p *Peer) {
	pm.peers = append(pm.peers, p)
	log.WithField("prefix", "peermgr").Infof("Adding peer %s into the Peer Manager", p.Conn.RemoteAddr().String())
}
