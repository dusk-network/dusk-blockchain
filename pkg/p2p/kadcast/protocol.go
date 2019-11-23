package kadcast

import (
	"errorss"
	"log"
	"time"
)

func initBootstrap(router Router, bootNodes []Peer) error {
	log.Println("Bootstrapping process started.")
	// Get PeerList ordered by distance so we can compare it
	// after the `PONG` arrivals.
	initPeerNum := router.tree.getTotalPeers()
	// Send `PING` to the bootstrap nodes.
	for _, peer := range bootNodes {
		router.sendPing(peer)
	}
	// Wait for `PONG` responses.
	time.Sleep(time.Second * 5)
	attempts := 0
	for {
		// If new peers were added (the bootstrap nodes)
		// we consider that the bootstrapping succeeded.
		if initPeerNum <= router.tree.getTotalPeers() {
			if attempts < 3 {
				log.Printf("Bootstrapping nodes were not added on attempt nº %v\nTrying again...\n", attempts)
				attempts++
			}
			return errors.New("Maximum number of attempts achieved. Please review yor connection settings.")
		}
		return nil
	}
}
