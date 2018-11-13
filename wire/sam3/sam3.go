package sam3

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// SAM control socket
type SAM struct {
	Address string   // IPv4:port
	Conn    net.Conn // Connection to the router
	Session Session  // Running session on this socket
}

// Default SAM bridge return messages
const (
	sessionDuplicateID   = "SESSION STATUS RESULT=DUPLICATED_ID\n"
	sessionDuplicateDest = "SESSION STATUS RESULT=DUPLICATED_DEST\n"
	sessionInvalidKey    = "SESSION STATUS RESULT=INVALID_KEY\n"
	sessionI2PError      = "SESSION STATUS RESULT=I2P_ERROR"
	sessionOK            = "SESSION STATUS RESULT=OK"
)

// NewSAM creates a new SAM control socket on the I2P router.
func NewSAM(address string) (*SAM, error) {
	conn, err := net.Dial("tcp4", address)
	if err != nil {
		return nil, err
	}

	// Allow up to SAMv3.3
	msg := []byte("HELLO VERSION MIN=3.0 MAX=3.3\n")
	text, err := SendToBridge(msg, conn)
	if err != nil {
		return nil, err
	}

	// Handle SAM bridge response
	if strings.Contains(text, "HELLO REPLY RESULT=OK VERSION=3") {
		return &SAM{
			Address: address,
			Conn:    conn,
		}, nil
	} else if text == "HELLO REPLY RESULT=NOVERSION\n" {
		return nil, errors.New("router does not support SAMv3")
	} else {
		return nil, errors.New(text)
	}
}

// NewKeys creates the I2P equivalent of an IP address.
// The public key is the destination (the public address),
// and the private key is used to send information and start a session with.
func (s *SAM) NewKeys() (I2PKeys, error) {
	// Default to ECDSA_SHA256_P256 over DSA_SHA1.
	msg := []byte("DEST GENERATE SIGNATURE_TYPE=ECDSA_SHA256_P256\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		s.Close()
		return I2PKeys{}, err
	}

	// Handle response
	fields := strings.Fields(text)[2:] // Omit "DEST REPLY"
	var keys I2PKeys
	for _, field := range fields {
		if strings.HasPrefix(field, "PUB=") {
			keys.Addr = field[4:]
		} else if strings.HasPrefix(field, "PRIV=") {
			keys.Priv = field[5:]
		} else {
			s.Close()
			return I2PKeys{}, errors.New("failed to parse keys")
		}
	}

	return keys, nil
}

// Lookup performs a lookup by going through the router's known and cached addresses
// and by querying it's peers.
func (s *SAM) Lookup(name string) (string, error) {
	msg := []byte("NAMING LOOKUP NAME=" + name + "\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		return "", err
	}

	// Handle response
	fields := strings.Fields(text)[2:] // Omit "NAMING REPLY"
	var addr string
	for _, field := range fields {
		switch {
		case field == "RESULT=OK":
			continue
		case field == "RESULT=INVALID_KEY":
			return "", errors.New("invalid key")
		case field == "RESULT=KEY_NOT_FOUND":
			return "", errors.New("unable to resolve " + name)
		case strings.HasPrefix(field, "VALUE="):
			addr = field[6:]
		case strings.HasPrefix(field, "MESSAGE="):
			return "", errors.New(field[8:])
		default:
			continue
		}
	}

	return addr, nil
}

// HandleResponse is a convenience function for handling standard SAM bridge responses
// when creating sessions. This should only be used for SAM session responses.
func (s *SAM) HandleResponse(text string) error {
	switch {
	case strings.HasPrefix(text, sessionOK):
		return nil
	case text == sessionDuplicateID:
		return errors.New("duplicate tunnel name")
	case text == sessionDuplicateDest:
		return errors.New("duplicate destination")
	case text == sessionInvalidKey:
		return errors.New("invalid key")
	case strings.HasPrefix(text, sessionI2PError):
		return fmt.Errorf("I2P error: %v", text[len(sessionI2PError):])
	default:
		return fmt.Errorf("unable to parse SAMv3 reply: %v", text)
	}
}

// Close this SAM session
func (s *SAM) Close() error {
	// Close running session, if any
	if s.Session != nil {
		if err := s.Session.Close(); err != nil {
			return err
		}
	} else {
		// Session will write EXIT to socket automatically on closing.
		// If no session is open, do it here.
		WriteMessage([]byte("EXIT"), s.Conn)
		if err := s.Conn.Close(); err != nil {
			return err
		}
	}

	return nil
}
