package sam3

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// StreamSession defines a SAM streaming session.
type StreamSession struct {
	ID      string        // Tunnel name
	Keys    I2PKeys       // I2P destination keys
	Conn    net.Conn      // Connection to SAM bridge
	Streams []*StreamConn // All open connections in the StreamSession go here
}

// NewStreamSession creates a new StreamSession on the SAM control socket.
func (s *SAM) NewStreamSession(id string, keys I2PKeys, SAMOpt []string, I2CPOpt []string) (*StreamSession, error) {
	// Check user options
	for _, opt := range SAMOpt {
		flag := strings.Split(opt, "=")[0]

		// Handle improper flags
		if flag == "PROTOCOL" || flag == "LISTEN_PROTOCOL" || flag == "PORT" ||
			flag == "HOST" || flag == "HEADER" {
			return nil, fmt.Errorf("invalid flag %v for a streaming session", flag)
		}
	}

	// Write SESSION CREATE message
	msg := []byte("SESSION CREATE STYLE=STREAM ID=" + id + " DESTINATION=" + keys.Priv + " " +
		strings.Join(SAMOpt, " ") + " " + strings.Join(I2CPOpt, " ") + "\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		return nil, err
	}

	if err := s.HandleResponse(text); err != nil {
		return nil, err
	}

	// Populate a StreamSession struct
	sess := StreamSession{
		ID:   id,
		Keys: keys,
		Conn: s.Conn,
	}

	// Add session to SAM
	s.Session = &sess
	return &sess, nil
}

// HandleResponse is a convenience method for handling stream session responses
// from the SAM control socket.
func (ss *StreamSession) HandleResponse(text string) error {
	result := strings.TrimPrefix(text, "STREAM STATUS ")
	switch result {
	case "RESULT=OK\n":
		return nil
	case "RESULT=CANT_REACH_PEER\n":
		return errors.New("can not reach peer")
	case "RESULT=I2P_ERROR\n":
		return errors.New("I2P internal error")
	case "RESULT=INVALID_KEY\n":
		return errors.New("invalid key")
	case "RESULT=INVALID_ID\n":
		return errors.New("invalid tunnel ID")
	case "RESULT=TIMEOUT\n":
		return errors.New("timeout")
	default:
		return errors.New("unknown error: " + result)
	}
}

// Close the stream session.
func (ss *StreamSession) Close() error {
	// Close all connections first. Since StreamConns reorganize the array
	// themselves, keep closing the StreamConn on the zero index
	// until a nil pointer is encountered.
	for {
		if len(ss.Streams) == 0 {
			break
		}

		if err := ss.Streams[0].Close(); err != nil {
			return err
		}
	}

	WriteMessage([]byte("EXIT"), ss.Conn)
	return ss.Conn.Close()
}

// StreamConn defines an open streaming connection between two parties.
type StreamConn struct {
	Session    *StreamSession // The StreamSession this connection is running on.
	LocalAddr  string         // Our destination
	RemoteAddr string         // Destination of the streaming counterparty
	Conn       net.Conn       // Connection to the SAM control socket
}

const acceptMessageLen = 4127 // Max key size + FROM_PORT/TO_PORT information

// Read from the StreamConn.
func (sc *StreamConn) Read(buf []byte) (int, error) {
	n, err := sc.Conn.Read(buf)
	return n, err
}

// Write to the StreamConn.
func (sc *StreamConn) Write(buf []byte) (int, error) {
	n, err := sc.Conn.Write(buf)
	return n, err
}

// Connect dials to an I2P destination and starts streaming with it if successful.
func (ss *StreamSession) Connect(addr string) (*StreamConn, error) {
	s, err := NewSAM(ss.Conn.RemoteAddr().String())

	msg := []byte("STREAM CONNECT ID=" + ss.ID + " DESTINATION=" + addr + " SILENT=false\n")
	text, err := SendToBridge(msg, s.Conn) // will return once the counterparty accepts or rejects
	if err != nil {
		s.Close()
		return nil, err
	}

	if err := ss.HandleResponse(text); err != nil {
		s.Close()
		return nil, err
	}

	return &StreamConn{
		Session:    ss,
		LocalAddr:  ss.Keys.Addr,
		RemoteAddr: addr,
		Conn:       s.Conn,
	}, nil
}

// Accept incoming connections to your streaming session.
func (ss *StreamSession) Accept(silent bool) (*StreamConn, error) {
	// Get new SAM socket for the StreamConn
	s, err := NewSAM(ss.Conn.RemoteAddr().String())

	// Send message to SAM
	msg := []byte("STREAM ACCEPT ID=" + ss.ID + " SILENT=" +
		strconv.FormatBool(silent) + "\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		return nil, err
	}

	if err := ss.HandleResponse(text); err != nil {
		return nil, err
	}

	var sc StreamConn

	// If SILENT=false, wait for the message containing the dest
	if !silent {
		// Read message once we get one
		buf := make([]byte, acceptMessageLen)
		if _, err := s.Conn.Read(buf); err != nil {
			return nil, err
		}

		// Handle whatever comes through the socket
		i := bytes.IndexByte(buf, '\n')
		if i == -1 {
			return nil, errors.New("SAM bridge message was not properly terminated")
		}

		data := string(buf[:i])
		dest := strings.Split(data, "FROM_PORT=")[0]

		// Create StreamConn
		sc = StreamConn{
			Session:    ss,
			Conn:       s.Conn,
			LocalAddr:  ss.Keys.Addr,
			RemoteAddr: dest,
		}
	} else {
		// If SILENT=true, just return a connection
		sc = StreamConn{
			Session:   ss,
			Conn:      s.Conn,
			LocalAddr: ss.Keys.Addr,
		}
	}

	// Add connection to the session list and return it
	ss.Streams = append(ss.Streams, &sc)
	return &sc, nil
}

// Forward incoming connections to another destination.
func (ss *StreamSession) Forward(dest string, port string) (*StreamConn, error) {
	// Get new SAM socket for the stream
	s, err := NewSAM(ss.Conn.RemoteAddr().String())

	// Listen on new SAM socket
	l, err := net.Listen("tcp4", s.Conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}

	// Send message to SAM
	msg := []byte("STREAM FORWARD ID=" + ss.ID + " PORT=" + port + " HOST=" + dest + " SILENT=false\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		return nil, err
	}

	if err := ss.HandleResponse(text); err != nil {
		l.Close()
		return nil, err
	}

	return &StreamConn{
		Session:   ss,
		Conn:      s.Conn,
		LocalAddr: ss.Keys.Addr,
	}, nil
}

// Close the StreamConn.
func (sc *StreamConn) Close() error {
	// Remove itself from the Streams array
	ss := sc.Session.Streams
	for i, conn := range ss {
		if conn == sc {
			ss[i] = nil
			if i != len(ss)-1 {
				ss[i], ss[len(ss)-1] = ss[len(ss)-1], ss[i]
			}

			ss = ss[:len(ss)-1]
			sc.Session.Streams = ss
		}
	}

	WriteMessage([]byte("EXIT"), sc.Conn)
	return sc.Conn.Close()
}
