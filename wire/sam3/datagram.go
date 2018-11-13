package sam3

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// The DatagramSession implements net.PacketConn. It can send and receive
// signed datagrams. The maximum size is 32KB. Note that around 427 bytes
// will be taken up by signature data.
type DatagramSession struct {
	ID       string       // Session name
	Keys     I2PKeys      // I2P keys
	Conn     net.Conn     // Connection to the SAM control socket
	UDPConn  *net.UDPConn // Used to deliver datagrams
	RUDPAddr *net.UDPAddr // The SAM control socket UDP address
	FromPort string       // FROM_PORT specified on creation
	ToPort   string       // TO_PORT specified on creation
}

// NewDatagramSession creates a new datagram session on the SAM bridge.
func (s *SAM) NewDatagramSession(id string, keys I2PKeys, SAMOpt []string, I2CPOpt []string) (*DatagramSession, error) {
	// Set defaults first
	udpPort := "7655" // Default SAM UDP port (FROM_PORT/LISTEN_PORT)
	sendPort := "0"   // Default send port (TO_PORT/PORT)
	lHost, _, err := net.SplitHostPort(s.Conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}

	rHost, _, err := net.SplitHostPort(s.Conn.RemoteAddr().String())
	if err != nil {
		return nil, err
	}

	// Check user options
	for _, opt := range SAMOpt {
		flag := strings.Split(opt, "=")[0]

		if flag == "PORT" || flag == "TO_PORT" {
			sendPort = strings.Split(opt, "=")[1]
			n, err := strconv.Atoi(sendPort)
			if err != nil {
				return nil, err
			}

			if n > 65535 || n < 0 {
				return nil, fmt.Errorf("invalid port %d specified, should be between 0-65535", n)
			}
		}

		if flag == "FROM_PORT" || flag == "LISTEN_PORT" {
			udpPort = strings.Split(opt, "=")[1]
			n, err := strconv.Atoi(udpPort)
			if err != nil {
				return nil, err
			}

			if n > 65535 || n < 0 {
				return nil, fmt.Errorf("invalid port %d specified, should be between 0-65535", n)
			}
		}

		if flag == "HOST" {
			lHost = strings.Split(opt, "=")[1]
		}

		// Handle improper flags
		if flag == "PROTOCOL" || flag == "LISTEN_PROTOCOL" || flag == "HEADER" {
			return nil, fmt.Errorf("invalid flag %v for a datagram session", flag)
		}
	}

	// Set up connections to populate session struct with
	lUDPAddr, err := net.ResolveUDPAddr("udp4", lHost+":"+sendPort)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp4", lUDPAddr)
	if err != nil {
		return nil, err
	}

	rUDPAddr, err := net.ResolveUDPAddr("udp4", rHost+":"+udpPort)
	if err != nil {
		return nil, err
	}

	// Write SESSION CREATE message
	msg := []byte("SESSION CREATE STYLE=DATAGRAM ID=" + id + " DESTINATION=" + keys.Priv + " " +
		strings.Join(SAMOpt, " ") + " " + strings.Join(I2CPOpt, " ") + "\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		s.Close()
		return nil, err
	}

	// Check for any returned errors
	if err := s.HandleResponse(text); err != nil {
		s.Close()
		return nil, err
	}

	sess := DatagramSession{
		ID:       id,
		Keys:     keys,
		Conn:     s.Conn,
		UDPConn:  udpConn,
		RUDPAddr: rUDPAddr,
		FromPort: udpPort,
		ToPort:   sendPort,
	}

	// Add session to SAM
	s.Session = &sess
	return &sess, nil
}

// ReadFrom reads one datagram sent to the destination of the DatagramSession.
func (s *DatagramSession) Read() ([]byte, string, string, string, error) {
	buf := make([]byte, 32768+4168) // Max datagram size + max SAM bridge message size.
	n, sAddr, err := s.UDPConn.ReadFromUDP(buf)
	if err != nil {
		return nil, "", "", "", err
	}

	// Only accept incoming UDP messages from the SAM socket we're connected to.
	if !bytes.Equal(sAddr.IP, s.RUDPAddr.IP) {
		return nil, "", "", "", fmt.Errorf("datagram received from wrong address: expected %v, actual %v",
			s.RUDPAddr.IP, sAddr.IP)
	}

	// Split message lines first
	i := bytes.IndexByte(buf, byte('\n'))
	msg, data := string(buf[:i]), buf[i+1:n]

	// Split message into fields
	fields := strings.Split(msg, " ")

	// Handle message
	var dest string
	fromPort := "0" // Default FROM_PORT
	toPort := "0"   // Default TO_PORT
	for _, field := range fields {
		switch {
		case strings.Contains(field, "DESTINATION="):
			dest = strings.TrimPrefix(field, "DESTINATION=")
		case strings.Contains(field, "FROM_PORT="):
			fromPort = strings.TrimPrefix(field, "FROM_PORT=")
		case strings.Contains(field, "TO_PORT="):
			toPort = strings.TrimPrefix(field, "TO_PORT=")
		default:
			continue // SIZE is not important as we could determine this from ReadFromUDP
		}
	}

	return data, dest, fromPort, toPort, nil
}

// WriteTo sends one signed datagram to the destination specified. At the time of
// writing, maximum size is 31 kilobyte, but this may change in the future.
func (s *DatagramSession) Write(b []byte, addr string) (int, error) {
	header := []byte("3.3 " + s.ID + " " + addr + " FROM_PORT=" + s.FromPort + " TO_PORT=" + s.ToPort + "\n")
	msg := append(header, b...)
	n, err := s.UDPConn.WriteToUDP(msg, s.RUDPAddr)

	return n, err
}

// Close the DatagramSession. Calling this function will also close the associated
// SAM socket, so make sure to call this only outside of a master session.
func (s *DatagramSession) Close() error {
	WriteMessage([]byte("EXIT"), s.Conn)
	if err := s.Conn.Close(); err != nil {
		return err
	}

	err := s.UDPConn.Close()
	return err
}
