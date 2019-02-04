package connlstnr

import (
	log "github.com/sirupsen/logrus"
	"net"
)

// connlstnr listens to events typically related to connections
type connlstnr struct {
	config Config
}

// New returns a Connection Listener
func New(cfg Config) *connlstnr {
	cl := &connlstnr{
		config: cfg,
	}
	return cl
}

// OnConnection is called when a successful outbound connection has been made
func (c *connlstnr) OnConnection(conn net.Conn, addr string) {
	log.WithField("prefix", "connmgr").Infof("A connection to node %s was created", addr)

	p := c.config.CreatePeer(conn, false)
	err := p.Run()

	if err != nil {
		log.WithField("prefix", "connmgr").Errorf("Failed to run peer: %s", err.Error())
		c.OnFail(p.RemoteAddr().String())
		p.Disconnect()
		return
	}

	c.config.ConnectionComplete(conn.RemoteAddr().String(), false)

	log.WithField("prefix", "connmgr").Infof("Connected successfully to peer %s", p.RemoteAddr().String())

	latestHdr, _ := c.config.GetLatestHeader()
	if err = p.RequestHeaders(latestHdr.Hash); err != nil {
		log.WithField("prefix", "connmgr").Errorf("Failed to request headers from peer %s: %s", addr, err.Error())
	}
}

// OnAccept is called when a successful inbound connection has been made
func (c *connlstnr) OnAccept(conn net.Conn) {
	log.WithField("prefix", "connmgr").Infof("Peer %s wants to connect", conn.RemoteAddr().String())

	p := c.config.CreatePeer(conn, true)

	err := p.Run()
	if err != nil {
		log.WithField("prefix", "connmgr").Errorf("Failed to run peer: %s", err.Error())
		c.OnFail(p.RemoteAddr().String())
		p.Disconnect()
		return
	}

	c.config.ConnectionComplete(conn.RemoteAddr().String(), true)

	log.WithField("prefix", "connmgr").Infof("Accepted peer %s successfully", p.RemoteAddr().String())
}

// GetAddress gets a new address from Address Manager
func (c *connlstnr) GetAddress() (string, error) {
	return c.config.NewAddr()
}

// OnFail is called when a connection failed
func (c *connlstnr) OnFail(addr string) {
	c.config.Failed(addr)
}

// OnMinGetAddr is called when the minimum of new addresses has exceeded
func (c *connlstnr) OnMinGetAddr() {
	if err := c.config.RequestAddresses(); err != nil {
		log.WithField("prefix", "connmgr").Error("Failed to get addresses from peer after exceeding limit")
	}
}
