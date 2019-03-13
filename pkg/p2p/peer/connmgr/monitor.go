package connmgr

import (
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
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
				c.NewRequest()
			case event.MinGetAddrPeersExcd:
				log.WithField("prefix", "connmgr").Warnf("Number of current peers is under its 'minGetAddr' minimum")
				//TODO: Check if this must be nested in if nil
				if c.config.OnMinGetAddr != nil {
					go c.config.OnMinGetAddr()
				}
				//case event.OutOfPeerAddr:
				//	log.WithField("prefix", "connmgr").Warnf("Failed to find peer address. Peer addresses ran out")
				//	go c.config.OnMinGetAddr()
			}
		}
	}
}

func (c *Connmgr) monitorThresholds() {
	if uint16(len(c.ConnectedList)) > config.EnvNetCfg.Peer.Max {
		evt := event.GetEvent(event.MaxPeersExcd)
		if evt.IsExecutable() {
			c.conevtch <- evt.Execute()
		}
	}
	if uint16(len(c.ConnectedList)) < config.EnvNetCfg.Peer.Min {
		evt := event.GetEvent(event.MinPeersExcd)
		if evt.IsExecutable() {
			c.conevtch <- evt.Execute()
		}
	}
	if uint16(len(c.ConnectedList)) < config.EnvNetCfg.Peer.MinGetAddr {
		evt := event.GetEvent(event.MinGetAddrPeersExcd)
		if evt.IsExecutable() {
			c.conevtch <- evt.Execute()
		}
	}
}
