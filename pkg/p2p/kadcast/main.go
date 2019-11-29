package kadcast
/*package main

import "log"
import "kadcast"

func main() {
	// Our node info.
	var port uint16 = 25519
	ip := [4]byte{185,216,252,121}
	router := makeRouter(ip, port)

	// Launch a listener for our node.
	go startUDPListener("udp", &router)

	// Create BootstrapNodes Peer structs
	var port1 uint16 = 25519
	ip1 := [4]byte{178,62,17,199}
	boot1 := makePeer(ip1, port1)

	var bootstrapNodes []Peer
	bootstrapNodes = append(bootstrapNodes[:], boot1)

	// Start Bootstrapping process.
	err := initBootstrap(router, bootstrapNodes)
	if err != nil {
		log.Fatal("Error during the Bootstrap Process. Job terminated.")
	}

	// Once the bootstrap succeeded, start the network discovery.
	startNetworkDiscovery(router)

	for {

	}
}*/
