// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package util

import (
	"net"
)

// GetOutboundIP will return the machine's external IP address.
// https://stackoverflow.com/a/37382208
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = conn.Close()
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
