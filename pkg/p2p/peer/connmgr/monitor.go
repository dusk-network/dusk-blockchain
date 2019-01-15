package connmgr

import (
	log "github.com/sirupsen/logrus"
	cfg "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr/event"
)

func (c *Connmgr) monEvtLoop() {
	for {
		select {
		case evt := <-c.conevtch:
			switch evt.GetType() {
			case event.MaxPeersExcd:
			// Drop a connection
			case event.MinPeersExcd:
				log.WithField("prefix", "connmgr").Warnf("Number of current peers is under its minimum")
				go c.NewRequest()
			case event.MinGetAddrPeersExcd:
				log.WithField("prefix", "connmgr").Warnf("Number of current peers is under its 'minGetAddr' minimum")
				go c.OnMinGetAddr()
				//case event.OutOfPeerAddr:
				//	log.WithField("prefix", "connmgr").Warnf("Failed to find peer address. Peer addresses ran out")
				//	go c.config.OnMinGetAddr()
			}
		}
	}
}

func (c *Connmgr) monitorThresholds() {
	if len(c.ConnectedList) > cfg.GetInt("net.peer.max") {
		evt := event.GetEvent(event.MaxPeersExcd)
		if evt.IsExecutable() {
			c.conevtch <- evt.Execute()
		}
	}
	if len(c.ConnectedList) < cfg.GetInt("net.peer.min") {
		evt := event.GetEvent(event.MinPeersExcd)
		if evt.IsExecutable() {
			c.conevtch <- evt.Execute()
		}
	}
	if len(c.ConnectedList) < cfg.GetInt("net.peer.mingetaddr") {
		evt := event.GetEvent(event.MinGetAddrPeersExcd)
		if evt.IsExecutable() {
			c.conevtch <- evt.Execute()
		}
	}
}
