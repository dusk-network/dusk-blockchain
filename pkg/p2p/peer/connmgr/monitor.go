package connmgr

import (
	log "github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr/event"
)

func (c *Connmgr) monEvtLoop() {
	for {
		select {
		case evt := <-c.conevtch:
			switch evt {
			case event.MaxPeersExcd:
			// Drop a connection
			case event.MinPeersExcd:
				log.WithField("prefix", "connmgr").Warnf("Number of current peers is under its minimum")
				go c.NewRequest()
				//case event.MinGetAddrPeersExcd:
				//	log.WithField("prefix", "connmgr").Warnf("Number of current peers dropped under its 'minGetAddr' minimum")
				//	go c.config.OnMinGetAddr()
				//case event.OutOfPeerAddr:
				//	log.WithField("prefix", "connmgr").Warnf("Failed to find peer address. Peer addresses ran out")
				//	go c.config.OnMinGetAddr()
			}
		}
	}
}

func (c *Connmgr) monitorThresholds() {
	if len(c.ConnectedList) > cnf.GetInt("net.peer.max") {
		c.conevtch <- event.MaxPeersExcd
	}
	if len(c.ConnectedList) < cnf.GetInt("net.peer.min") {
		c.conevtch <- event.MinPeersExcd
	}
	if len(c.ConnectedList) < cnf.GetInt("net.peer.mingetaddr") {
		c.conevtch <- event.MinGetAddrPeersExcd
	}
}
