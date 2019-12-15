package kadcast

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
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
	for i := 0; i <= 3; i++ {

		actualPeers := router.pollBootstrappingNodes(bootNodes, time.Second*5)
		if actualPeers <= initPeerNum {
			if i == 3 {
				return errors.New("Maximum number of attempts achieved. Please review yor connection settings")
			}
			log.WithFields(log.Fields{
				"Retries": i,
			}).Warn("Bootstrapping nodes were not added.\nTrying again..")
		} else {
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
func StartNetworkDiscovery(router *Router) {
	// Get closest actual Peer.
	previousClosestArr := router.getXClosestPeersTo(1, router.MyPeerInfo)
	previousClosest := previousClosestArr[0]

	// Ask for new peers, wait for `PONG` arrivals and get the
	// new closest `Peer`.
	actualClosest := router.pollClosestPeer(5 * time.Second)

	// Until we don't get a peer closer to our node on each poll,
	// we look for more nodes.
	for actualClosest != previousClosest {
		previousClosest = actualClosest
		actualClosest = router.pollClosestPeer(5 * time.Second)
	}

	log.WithFields(log.Fields{
		"peers_connected": router.tree.getTotalPeers(),
	}).Info("Network Discovery process finished.")
}
