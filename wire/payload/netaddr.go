package payload

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/toghrulmaharramov/dusk-go/encoding"
)

// NetAddress holds an IP and a port number of a Dusk network peer.
type NetAddress struct {
	IP   net.IP
	Port uint16
}

// Encode a NetAddress to w.
func (n *NetAddress) Encode(w io.Writer) error {
	var ip [16]byte
	if n.IP != nil {
		copy(ip[:], n.IP.To16())
	}

	if err := binary.Write(w, binary.LittleEndian, ip); err != nil {
		return err
	}

	if err := encoding.PutUint16(w, binary.LittleEndian, n.Port); err != nil {
		return err
	}

	return nil
}

// Decode a NetAddress from r.
func (n *NetAddress) Decode(r io.Reader) error {
	var ip [16]byte
	if err := binary.Read(r, binary.LittleEndian, ip); err != nil {
		return err
	}

	n.IP = net.IP(ip[:])

	port, err := encoding.Uint16(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	n.Port = port
	return nil
}
