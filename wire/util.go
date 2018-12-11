package wire

import (
	"net"
)

// GetLocalIP will return the machine's external IP address as a string.
// https://stackoverflow.com/a/37382208
func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}

	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
