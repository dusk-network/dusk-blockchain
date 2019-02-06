package connmgr

import (
	"net"
)

// Config contains all functions which will be set by the caller to setup the connection manager
type Config struct {

	// GetAddress will return a single address for the connection manager to connect to
	GetAddress func() (string, error)

	// OnConnection is called when we successfully connect to a peer
	OnConnection func(conn net.Conn, addr string)

	// OnAccept will take an established connection
	OnAccept func(net.Conn)

	// OnFail is called when a connection failed
	OnFail func(string)

	// OnMinGetAddr is called when the minimum of new addresses has exceeded
	OnMinGetAddr func()
}
