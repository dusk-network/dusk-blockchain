package main

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// CmgrConfig is the config file for the node connection manager
type CmgrConfig struct {
	Port     string
	OnAccept func(net.Conn)
	OnConn   func(net.Conn, string) // takes the connection  and the string
}

type connmgr struct {
	CmgrConfig
}

//NewConnMgr creates a new connection manager
func NewConnMgr(cfg CmgrConfig) *connmgr {
	cnnmgr := &connmgr{
		cfg,
	}

	go func() {
		addrPort := ":" + cfg.Port
		listener, err := net.Listen("tcp", addrPort)
		if err != nil {
			panic(err)
		}

		defer func() {
			listener.Close()
		}()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.WithFields(log.Fields{
					"process": "connection manager",
					"error":   err,
				}).Warnln("error accepting connection request")
				continue
			}

			go cfg.OnAccept(conn)
		}
	}()

	return cnnmgr
}

// Connect dials a connection with its string, then on succession
// we pass the connection and the address to the OnConn method
func (c *connmgr) Connect(addr string) error {
	conn, err := c.Dial(addr)
	if err != nil {
		return err
	}

	if c.CmgrConfig.OnConn != nil {
		go c.CmgrConfig.OnConn(conn, addr)
	}

	return nil
}

// Dial dials up a connection, given its address string
func (c *connmgr) Dial(addr string) (net.Conn, error) {
	dialTimeout := 1 * time.Second
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
