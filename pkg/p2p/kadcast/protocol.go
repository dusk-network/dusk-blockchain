package kadcast

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
)

var isBootstrapping = false
var isDiscoveringNetwork = false

// InitBootstrap inits the Bootstrapping process by sending
// a `PING` message to every bootstrapping node repeatedly.
// If it tried 3 or more times and no new `Peers` were added,
// it panics.
// Otherways, it returns `nil` and logs the Number of peers
// the node is connected to at the end of the process.
func InitBootstrap(router *Router, bootNodes []Peer, wg *sync.WaitGroup) error {
	isBootstrapping = true
	log.Info("Bootstrapping process started.")
	// Get PeerList ordered by distance so we can compare it
	// after the `PONG` arrivals.
	initPeerNum := router.tree.getTotalPeers()
	for i := 0; i <= 3; i++ {
		// Send `PING` to the bootstrap nodes.
		for _, peer := range bootNodes {
			router.sendPing(peer)
		}
		// Wait for `PONG` responses.
		// With one of them we have enough.
		wg.Add(1)
		wg.Wait()
		// If new peers were added (the bootstrap nodes)
		// we consider that the bootstrapping succeeded.
		actualPeers := router.tree.getTotalPeers()
		if actualPeers <= initPeerNum {
			if i == 3 {
				isBootstrapping = false
				return errors.New("\nMaximum number of attempts achieved. Please review yor connection settings\n")
			}
			log.WithFields(log.Fields{
				"Tries": i,
			}).Warn("Bootstrapping nodes were not added.\nTrying again..")
		} else {
			isBootstrapping = false
			break
		}
	}
	log.WithFields(log.Fields{
		"connected_nodes": router.tree.getTotalPeers(),
	}).Info("Bootstrapping process finished")
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
func StartNetworkDiscovery(router *Router, wg *sync.WaitGroup) {
	isDiscoveringNetwork = true
	var actualClosest []Peer
	previousClosest := router.getXClosestPeersTo(1, router.MyPeerInfo)
	// Ask for nodes to `alpha` closest nodes to my peer.
	router.sendFindNodes()
	// Wait until response arrives and we query the nodes.
	wg.Add(1)
	wg.Wait()
	for {
		actualClosest = router.getXClosestPeersTo(1, router.MyPeerInfo)
		if actualClosest[0] == previousClosest[0] {
			log.WithField(
				"connected_peers", router.tree.getTotalPeers(),
			).Info("Network Discovery process has finished!")
			isDiscoveringNetwork = false
			return
		}
		// We get the closest actual Peer.
		previousClosest = actualClosest
		// Send `FIND_NODES` again.
		router.sendFindNodes()
		// Wait until response arrives and we query the nodes.
		wg.Add(1)
		wg.Wait()
	}
}
