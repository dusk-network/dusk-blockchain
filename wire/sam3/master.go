package sam3

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// MasterSession allows the user to set up a master session, with the ability to run
// any of the 3 protocols (datagram, raw, stream) simultaneously on the same socket.
type MasterSession struct {
	ID       string             // Session ID
	Keys     I2PKeys            // I2P keys
	Conn     net.Conn           // Connection to the SAM bridge
	Sessions map[string]Session // Maps session IDs to their structs for easy access
	SIDs     []string           // All sessions on the master session by ID
}

// Options that can't be passed to a master session on creation.
var invalidOpt = map[string]bool{
	"PORT":            true,
	"HOST":            true,
	"FROM_PORT":       true,
	"TO_PORT":         true,
	"PROTOCOL":        true,
	"LISTEN_PORT":     true,
	"LISTEN_PROTOCOL": true,
}

// Session is implemented by any session struct in this package.
// It's intended for record-keeping purposes on the MasterSession struct,
// and shouldn't be used for anything else, as most other structs also implement
// a Close method.
type Session interface {
	Close() error
}

// NewMasterSession creates a new master session on the SAM bridge.
func (s *SAM) NewMasterSession(id string, keys I2PKeys, I2CPOpt []string) (*MasterSession, error) {
	// Don't allow any options other than I2CP options.
	for _, opt := range I2CPOpt {
		flag := strings.Split(opt, "=")[0]
		if invalidOpt[flag] {
			return nil, fmt.Errorf("master session options may not contain %v", flag)
		}
	}

	// Format session creation message
	msg := []byte("SESSION CREATE STYLE=MASTER ID=" + id + " DESTINATION=" + keys.Priv + " " +
		strings.Join(I2CPOpt, " ") + "\n")

	// Write message to SAM bridge
	if err := WriteMessage(msg, s.Conn); err != nil {
		return nil, err
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := s.Conn.Read(buf)
	if err != nil {
		s.Close()
		return nil, err
	}

	// Check for any returned errors
	text := string(buf[:n])
	if err := s.HandleResponse(text); err != nil {
		s.Close()
		return nil, err
	}

	master := MasterSession{
		ID:       id,
		Keys:     keys,
		Conn:     s.Conn,
		Sessions: make(map[string]Session),
	}

	s.Session = &master
	return &master, nil
}

// Add a subsession on the master session.
func (s *MasterSession) Add(style string, id string, SAMOpt []string) (Session, error) {
	// Make sure DESTINATION isn't passed
	for _, opt := range SAMOpt {
		flag := strings.Split(opt, "=")[0]
		if flag == "DESTINATION" {
			return nil, errors.New("subsession may not contain DESTINATION flag")
		}
	}

	// Make sure style is in uppercase and handle accordingly
	su := strings.ToUpper(style)
	switch su {
	case "RAW":
		// Set defaults first
		udpPort := "7655" // Default SAM UDP port (FROM_PORT/LISTEN_PORT)
		sendPort := "0"   // Default send port (TO_PORT/PORT)
		protocol := "18"  // Default protocol for raw sessions
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

			// If passed, verify protocol.
			if flag == "PROTOCOL" || flag == "LISTEN_PROTOCOL" {
				protocol = strings.Split(opt, "=")[1]
				pInt, err := strconv.Atoi(protocol)
				if err != nil {
					return nil, err
				}

				// Check if it's within bounds, and make sure it's not specified as streaming protocol.
				if pInt < 0 || pInt > 255 || pInt == 6 {
					return nil, fmt.Errorf("Bad RAW LISTEN_PROTOCOL %d", pInt)
				}
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

		// Write SESSION ADD message
		msg := []byte("SESSION ADD STYLE=RAW ID=" + id + " PORT=" + sendPort + " " +
			strings.Join(SAMOpt, " ") + "\n")
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

		// Populate master session and create RawSession struct
		s.SIDs = append(s.SIDs, id)
		sess := RawSession{
			ID:       id,
			Keys:     s.Keys,
			Conn:     s.Conn,
			UDPConn:  udpConn,
			RUDPAddr: rUDPAddr,
			FromPort: udpPort,
			ToPort:   sendPort,
			Protocol: protocol,
		}

		s.Sessions[id] = &sess
		return &sess, nil
	case "DATAGRAM":
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

		// Write SESSION ADD message
		msg := []byte("SESSION ADD STYLE=DATAGRAM ID=" + id + " PORT=" + sendPort + " " +
			strings.Join(SAMOpt, " ") + "\n")
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

		// Populate master session and create RawSession struct
		s.SIDs = append(s.SIDs, id)
		sess := DatagramSession{
			ID:       id,
			Keys:     s.Keys,
			Conn:     s.Conn,
			UDPConn:  udpConn,
			RUDPAddr: rUDPAddr,
			FromPort: udpPort,
			ToPort:   sendPort,
		}

		s.Sessions[id] = &sess
		return &sess, nil
	case "STREAM":
		// Check user options
		for _, opt := range SAMOpt {
			flag := strings.Split(opt, "=")[0]

			// Handle improper flags
			if flag == "PROTOCOL" || flag == "LISTEN_PROTOCOL" || flag == "PORT" ||
				flag == "HOST" || flag == "HEADER" {
				return nil, fmt.Errorf("invalid flag %v for a streaming session", flag)
			}
		}

		// Write SESSION ADD message
		msg := []byte("SESSION ADD STYLE=STREAM ID=" + id + " " + strings.Join(SAMOpt, " ") + "\n")
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

		// Populate master session and create RawSession struct
		s.SIDs = append(s.SIDs, id)
		sess := StreamSession{
			ID:   id,
			Keys: s.Keys,
			Conn: s.Conn,
		}

		s.Sessions[id] = &sess
		return &sess, nil
	default:
		return nil, fmt.Errorf("session style %v not recognized by SAM", su)
	}
}

// Remove a subsession from the master session.
func (s *MasterSession) Remove(id string) error {
	// Write SESSION REMOVE message
	msg := []byte("SESSION REMOVE ID=" + id + "\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		s.Close()
		return err
	}

	if err := s.HandleResponse(text); err != nil {
		s.Close()
		return err
	}

	// Remove subsession from map
	s.Sessions[id] = nil

	// Remove subsession from array, and moving elements to keep the slice in one piece.
	// This is done to ensure we properly remove all sessions when calling Close.
	for i, sess := range s.SIDs {
		if sess == id {
			s.SIDs[i] = "" // Clear ID

			// Move element to end of slice
			if i != len(s.SIDs)-1 {
				s.SIDs[i], s.SIDs[len(s.SIDs)-1] = s.SIDs[len(s.SIDs)-1], s.SIDs[i]
			}

			// Cut the element out of the slice
			s.SIDs = s.SIDs[:len(s.SIDs)-1]
		}
	}

	return nil
}

// HandleResponse is a convenience function for handling SAMv3 responses when creating subsessions.
func (s *MasterSession) HandleResponse(text string) error {
	switch {
	case text == sessionDuplicateID:
		return errors.New("duplicate tunnel name")
	case text == sessionDuplicateDest:
		return errors.New("duplicate destination")
	case text == sessionInvalidKey:
		return errors.New("invalid key")
	case strings.HasPrefix(text, sessionI2PError):
		return fmt.Errorf("I2P error: %v", text[len(sessionI2PError):])
	case strings.HasPrefix(text, sessionOK):
		return nil
	default:
		return fmt.Errorf("unable to parse SAMv3 reply: %v", text)
	}
}

// Close closes the master session and the corresponding I2P session.
func (s *MasterSession) Close() error {
	if len(s.SIDs) > 0 {
		// Close all subsessions first
		for {
			// Keep deleting on zero index until the slice is empty
			if len(s.SIDs) == 0 {
				break
			}

			if err := s.Remove(s.SIDs[0]); err != nil {
				WriteMessage([]byte("EXIT"), s.Conn)
				s.Conn.Close()
				return err
			}
		}
	}

	// Close connection to SAM bridge.
	WriteMessage([]byte("EXIT"), s.Conn)
	err := s.Conn.Close()
	return err
}
