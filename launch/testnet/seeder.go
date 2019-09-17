package main

import (
	"net"
	"os"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	cli "github.com/dusk-network/dusk-blockchain/pkg/voucher/client/utils"
	log "github.com/sirupsen/logrus"
)

// ConnectToSeeder initializes the connection with the Voucher Seeder
func ConnectToSeeder() []string {
	if cfg.Get().General.Network == "testnet" {
		fixedNetwork := cfg.Get().Network.Seeder.Fixed
		if len(fixedNetwork) > 0 {
			log.Infof("Fixed-network config activated")
			return fixedNetwork
		}
	}

	seeders := cfg.Get().Network.Seeder.Addresses
	if len(seeders) == 0 {
		log.Errorf("Empty list of seeder addresses")
		return nil
	}
	seeder := seeders[0]

	// Define the node information to submit to the voucher
	seederKey := []byte(os.Getenv("SEEDER_KEY"))
	nodePubKey := []byte("PubKey won't precede wallet")
	nodePort := ":" + cfg.Get().Network.Port
	nodeSocket, err := net.ResolveTCPAddr("tcp", nodePort)
	if err != nil {
		log.Errorf("Could not define the node network addr: %s", err)
		return nil
	}

	// Perform the request for the seeders
	nodes, err := cli.RequestSeeders(&seederKey, &seeder, &nodePubKey, nodeSocket)
	if err != nil {
		log.Errorf("Could not fetch the seeders from the voucher: %s", err)
		return nil
	}

	// Assemble the list of addresses from the nodes
	addrs := []string{}
	for _, node := range *nodes {
		addrs = append(addrs, node.Addr.String())
	}

	// start a goroutine with a ping loop for the seeder, so it knows when we shut down
	go func() {
		for {
			time.Sleep(4 * time.Second)
			err := cli.PingVoucher(&seeder)
			if err != nil {
				log.WithFields(log.Fields{
					"process": "main",
					"error":   err,
				}).Warnln("error pinging voucher seeder")
				return
			}
		}
	}()

	return addrs
}
