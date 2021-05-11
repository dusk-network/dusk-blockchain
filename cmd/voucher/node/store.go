// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package node

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Node defines the status of a Dusk node.
type Node struct {
	online          bool
	blackListedTime int64
	Challenge       []byte
	listeningPort   string
}

// max amount of ips we send to requesting nodes.
var maxIPs = 10

// Store contains a map to pointers to node elements, where their ip is used as key.
type Store struct {
	sync.RWMutex
	Nodes map[string]*Node
}

// NewStore creates a new instance of Store.
func NewStore() *Store {
	return &Store{
		Nodes: make(map[string]*Node),
	}
}

// Add a node to the list.
func (d *Store) Add(ip string, challenge []byte) {
	d.Lock()
	defer d.Unlock()

	d.Nodes[ip] = &Node{
		online:          true,
		blackListedTime: 0,
		Challenge:       challenge,
	}
}

// Get a node from the list.
func (d *Store) Get(ip string) Node {
	d.RLock()
	defer d.RUnlock()

	return *d.Nodes[ip]
}

// SetPort will the listeningPort on a node, which is used when sending addresses to
// other nodes.
func (d *Store) SetPort(ip, port string) {
	d.Lock()
	defer d.Unlock()

	node, ok := d.Nodes[ip]
	if !ok {
		return
	}

	node.listeningPort = port
}

// BlackList sets the blacklist flag on the node with the provided ip.
func (d *Store) BlackList(ip string) {
	d.Lock()
	defer d.Unlock()

	node, ok := d.Nodes[ip]
	if !ok {
		return
	}

	node.blackListedTime = time.Now().Unix()
}

// IsBlackListed checks if a node is present in the blacklist.
func (d *Store) IsBlackListed(ip string) bool {
	d.RLock()
	defer d.RUnlock()

	node, ok := d.Nodes[ip]
	if !ok || time.Now().Unix()-node.blackListedTime > 86400 {
		return false
	}

	return true
}

// DumpNodes returns a comma separated string of non-blacklisted, active IPs
// of dusk nodes currently stored.
func (d *Store) DumpNodes(srcPeerID string, _ message.Message) ([]bytes.Buffer, error) {
	ips := make([]bytes.Buffer, 0, maxIPs)

	d.RLock()
	defer d.RUnlock()

	for ip, n := range d.Nodes {
		if !d.IsBlackListed(ip) && strings.TrimSpace(ip) != srcPeerID &&
			n.online {
			buf := bytes.NewBuffer([]byte(d.getListeningAddr(ip)))
			if err := topics.Prepend(buf, topics.Addr); err != nil {
				return nil, err
			}

			ips = append(ips, *buf)
		}

		if len(ips) >= maxIPs {
			break
		}
	}

	return ips, nil
}

// SetInactive will set a node's status to down. Should be called after
// definitive disconnect.
func (d *Store) SetInactive(ip string) {
	d.Lock()
	defer d.Unlock()

	node, ok := d.Nodes[ip]
	if ok {
		node.online = false
	}
}

func (d *Store) getListeningAddr(ip string) string {
	d.RLock()
	defer d.RUnlock()

	node := d.Nodes[ip]
	trimmedIP := strings.Split(ip, ":")[0]
	return trimmedIP + ":" + node.listeningPort
}
