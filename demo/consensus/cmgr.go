package main

import (
	"fmt"
	"net"
	"time"
)

type CmgrConfig struct {
	Port     string
	OnAccept func(net.Conn)
	OnConn   func(net.Conn, string) // takes the connection  and the string
}

type connmgr struct {
	CmgrConfig
}

//New creates a new connection manager
func newConnMgr(cfg CmgrConfig) *connmgr {
	cnnmgr := &connmgr{
		cfg,
	}

	go func() {

		addrPort := "127.0.0.1" + ":" + cfg.Port

		listener, err := net.Listen("tcp", addrPort)

		if err != nil {
			fmt.Println("Error connecting to outbound ", err)
		}

		defer func() {
			listener.Close()
		}()

		for {

			conn, err := listener.Accept()

			if err != nil {
				continue
			}

			go cfg.OnAccept(conn)
		}

	}()

	return cnnmgr
}

// Connect dials a connection with it's string, then on succession
// we pass the connection and the address to the OnConn method
func (c *connmgr) Connect(addr string) error {

	conn, err := c.Dial(":" + addr)
	if err != nil {
		return err
	}

	if c.CmgrConfig.OnConn != nil {

		c.CmgrConfig.OnConn(conn, addr)
	}

	return nil

}

// Dial dials up a connection, given it's address string
func (c *connmgr) Dial(addr string) (net.Conn, error) {
	dialTimeout := 1 * time.Second
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
