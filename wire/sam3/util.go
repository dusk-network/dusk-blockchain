package sam3

import (
	"errors"
	"net"
)

// WriteMessage writes msg to the specified conn (which should be the SAM control socket)
func WriteMessage(msg []byte, conn net.Conn) error {
	for i, j := 0, 0; i != len(msg); j++ {
		// Stop after 15 tries
		if j >= 15 {
			return errors.New("write to SAM bridge failed")
		}

		// Attempt to write msg
		n, err := conn.Write(msg[i:])
		if err != nil {
			return err
		}

		i += n
	}

	return nil
}

// SendToBridge sends msg to the SAM bridge and returns the reply.
func SendToBridge(msg []byte, conn net.Conn) (string, error) {
	if err := WriteMessage(msg, conn); err != nil {
		return "", err
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}

	return string(buf[:n]), nil
}
