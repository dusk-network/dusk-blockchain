// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"strings"
	"sync"
	"time"
)

// Node defines the status of a Dusk node
type Node struct {
	status          connStatus
	lastSeen        string
	blackListed     bool
	blackListedTime string
}

type connStatus int

const (
	active connStatus = 1
	down   connStatus = 0
)

// max amount of ips we send to requesting nodes
var maxIPs = 10

// DuskNodes contains a map to pointers to node elements, where their ip is used as key.
type DuskNodes struct {
	sync.RWMutex
	Nodes map[string]*Node
}

// New creates a new instance of DuskNodes
func New() *DuskNodes {
	return &DuskNodes{
		Nodes: make(map[string]*Node),
	}
}

// BlackList sets the blacklist flag on the node with the provided ip
func (d *DuskNodes) BlackList(ip string, flag bool) {
	node := d.Get(ip)
	bltime := ""
	// reset the blacklist timer
	if flag {
		bltime = time.Now().Format(time.RFC3339)
	}

	if node == nil {
		node = &Node{
			status:          1,
			lastSeen:        time.Now().Format(time.RFC3339),
			blackListed:     flag,
			blackListedTime: bltime,
		}
	} else {
		node.blackListed = flag
		node.blackListedTime = bltime
	}

	d.Lock()
	defer d.Unlock()
	d.Nodes[ip] = node
}

// IsBlackListed checks if a node is present in the blacklist
func (d *DuskNodes) IsBlackListed(ip string) bool {
	node := d.Get(ip)

	if node == nil || !node.blackListed {
		return false
	}
	return true
}

// Update updates the DuskNodes struct with the connecting node
func (d *DuskNodes) Update(ip string) *Node {
	node := d.Get(ip)

	if node == nil {
		node = &Node{
			status:      active,
			lastSeen:    time.Now().Format(time.RFC3339),
			blackListed: false,
		}
	} else {
		node.status = active
		node.lastSeen = time.Now().Format(time.RFC3339)
	}
	d.Lock()
	defer d.Unlock()
	d.Nodes[ip] = node
	return node
}

// Get returns a node if present in the internal node list
func (d *DuskNodes) Get(ip string) *Node {
	d.RLock()
	defer d.RUnlock()
	if node, ok := d.Nodes[ip]; ok {
		return node
	}
	return nil
}

// CheckUnBAN checks if the node has been banned for more than 24 hours.
func (d *DuskNodes) CheckUnBAN(ip string) (bool, error) {
	node := d.Get(ip)

	if node == nil {
		// Node does not exist, nothing to unban
		return true, nil
	}

	if node.blackListed {
		bantime, e := time.Parse(time.RFC3339, node.blackListedTime)
		if e != nil {
			return false, e
		}
		duration := time.Since(bantime)
		if duration.Hours() > 24 {
			return true, nil
		}

	}
	return false, nil

}

// DumpNodes returns a comma separated string of non-blacklisted, active IPs
// of dusk nodes currently stored.
func (d *DuskNodes) DumpNodes(connectedIP string) string {

	ips := make([]string, 0, 1)
	d.RLock()
	defer d.RUnlock()
	for ip, n := range d.Nodes {
		if !n.blackListed && strings.TrimSpace(ip) != connectedIP &&
			n.status == active {
			ips = append(ips, strings.TrimSpace(ip))
		}

		if len(ips) >= maxIPs {
			break
		}
	}

	return strings.Join(ips, ",")
}

// SetInactive will set a node's status to down. Should be called after
// definitive disconnect.
func (d *DuskNodes) SetInactive(ip string) {
	node := d.Get(ip)
	if node != nil {
		node.status = down
		d.Lock()
		d.Nodes[ip] = node
		d.Unlock()
	}
}
