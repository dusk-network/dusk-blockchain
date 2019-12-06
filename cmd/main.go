package main

import (
	log "github.com/sirupsen/logrus"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

func main() {
	log.Infoln("Starting Kadcast Node!")
	// Our node info.
	var port uint16 = 25519
	ip := [4]byte{62, 57, 180, 247}
	router := kadcast.MakeRouter(ip, port)
	log.Infoln("Router was created Successfully.")

	// Create buffer.
	queue := ring.NewBuffer(500)
	// Create waitGroup
	var wg sync.WaitGroup

	// Launch PacketProcessor rutine.
	go kadcast.ProcessPacket(queue, &router, &wg)

	// Launch a listener for our node.
	go kadcast.StartUDPListener("udp", queue, router.MyPeerInfo)

	// Create BootstrapNodes Peer structs
	var port1 uint16 = 25519
	ip1 := [4]byte{157, 230, 219, 77}
	boot1 := kadcast.MakePeer(ip1, port1)
	var bootstrapNodes []kadcast.Peer
	bootstrapNodes = append(bootstrapNodes[:], boot1)

	// Start Bootstrapping process.
	err := kadcast.InitBootstrap(&router, bootstrapNodes, &wg)
	if err != nil {
		log.Panic("Error during the Bootstrap Process. Job terminated.")
	}

	// Once the bootstrap succeeded, start the network discovery.
	kadcast.StartNetworkDiscovery(&router, &wg)

	select {}
}
