package kadcast

import (
	"errors"
	"log"
	"time"
)

func initBootstrap(router *Router, bootNodes []Peer) error {
	log.Println("Bootstrapping process started.")
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
			if i < 3 {
				log.Printf("Bootstrapping nodes were not added on attempt nº %v\nTrying again...\n", i)
			} else {
				return errors.New("\nMaximum number of attempts achieved. Please review yor connection settings\n")
			}

		} else {
			break
		}
	}
	log.Printf("Bootstrapping process finnished. \nYou're now connected to %v nodes", router.tree.getTotalPeers())
	return nil
}

func startNetworkDiscovery(router *Router) {
	var actualClosest []Peer
	previousClosest := router.getXClosestPeersTo(1, router.myPeerInfo)
	// Ask for nodes to `alpha` closest nodes to my peer.
	router.sendFindNodes()
	// Wait until response arrives and we query the nodes.
	time.Sleep(time.Second * 5)
	for {
		actualClosest = router.getXClosestPeersTo(1, router.myPeerInfo)
		if actualClosest[0] == previousClosest[0] {
			log.Printf("Network Discovery process has finnished!.\nYou're now connected to %v", router.tree.getTotalPeers())
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
