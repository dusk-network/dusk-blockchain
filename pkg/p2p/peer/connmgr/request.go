package connmgr

import (
	"net"
)

// Request holds all that's needed to read and write messages from/to an address.
// It keeps count of the connection retries
type Request struct {
	Conn      net.Conn
	Addr      string
	Permanent bool
	Inbound   bool
	Retries   uint8 // should not be trying more than 255 tries
}
