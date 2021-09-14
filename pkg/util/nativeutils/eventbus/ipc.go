// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"net"
)

// IPCListener is an implementation of eventbus.Listener that allows processes
// to connect to a bus over a duplex serial connection (pipe, socket, tcp, etc)
type IPCListener struct {
	net.Conn
}

// NewIPCListener initialises and attaches a listener over a net.Conn with a
// mapper function to route received messages
func NewIPCListener(
	c net.Conn,
	mapper func(topic topics.Topic) byte,
) *IPCListener {
	il := &IPCListener{Conn: c}
	// attach the gRPC handler to the connection
	il.attach()
	// send out topic subscription request
	_ = mapper
	return il
}

func (il *IPCListener) attach() {

}

func (il *IPCListener) detach() {

}

// Notify passes a message to be relayed by the attached eventbus
func (il *IPCListener) Notify(message message.Message) error {
	// dispatch message to eventbus
	_ = message
	return nil
}

// Close the connection and stop receiving Notify calls
func (il *IPCListener) Close() {
	// close the gRPC
	il.detach()
}
