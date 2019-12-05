package kadcast

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"time"
)

// InitBootstrap inits the Bootstrapping process by sending 
// a `PING` message to every bootstrapping node repeatedly.
// If it tried 3 or more times and no new `Peers` were added,
// it panics. 
// Otherways, it returns `nil` and logs the Number of peers 
// the node is connected to at the end of the process.
func InitBootstrap(router *Router, bootNodes []Peer) error {
	log.Info("Bootstrapping process started.")
	// Get PeerList ordered by distance so we can compare it
	// after the `PONG` arrivals.
	initPeerNum := router.tree.getTotalPeers()
	for i := 0; i < 3; i++ {
		// Send `PING` to the bootstrap nodes.
		for _, peer := range bootNodes {
			router.sendPing(peer)
		}
		// Wait for `PONG` responses.
		time.Sleep(time.Second * 5)
		// If new peers were added (the bootstrap nodes)
		// we consider that the bootstrapping succeeded.
		actualPeers := router.tree.getTotalPeers()
		if actualPeers <= initPeerNum {
			if i >= 3 {
				return errors.New("\nMaximum number of attempts achieved. Please review yor connection settings\n")	
			} 
			log.Warning("Bootstrapping nodes were not added on attempt nº %v\nTrying again...\n", i)
		}
		break
	}
	log.Info("Bootstrapping process finished. \nYou're now connected to %v nodes", router.tree.getTotalPeers())
	return nil
}

// StartNetworkDiscovery triggers the network discovery process.
// The node basically sends `FIND_NODES` messages to the nodes it
// is currently connected to and evaluates the `Peers` that were added 
// on each iteration. 
// If the closest peer to ours is the same during two iterations of the
// `FIND_NODES` message, we finish the process logging the ammout of peers
// we are currently connected to.
// Otherways, if the closest Peer on two consecutive iterations changes, we 
// keep queriyng the `alpha` closest nodes with `FIND_NODES` messages.
func StartNetworkDiscovery(router *Router) {
	var actualClosest []Peer
	previousClosest := router.getXClosestPeersTo(1, router.MyPeerInfo)
	// Ask for nodes to `alpha` closest nodes to my peer.
	router.sendFindNodes()
	// Wait until response arrives and we query the nodes.
	time.Sleep(time.Second * 5)
	for {
		actualClosest = router.getXClosestPeersTo(1, router.MyPeerInfo)
		if actualClosest[0] == previousClosest[0] {
			log.Info("Network Discovery process has finished!.\nYou're now connected to %v", router.tree.getTotalPeers())
			return
		}
		// We get the closest actual Peer.
		previousClosest = actualClosest
		// Send `FIND_NODES` again.
		router.sendFindNodes()
		// Wait until response arrives and we query the nodes.
		time.Sleep(time.Second * 15)
	}
}
