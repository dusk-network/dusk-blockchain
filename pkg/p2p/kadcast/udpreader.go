// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rcudp"
)

const (
	redundancyFactor = uint8(2)
)

// RaptorCodeReader is rc-udp based listener that reads Broadcast messages from
// the Kadcast network and delegates their processing to the messageRouter.
type RaptorCodeReader struct {
	base        *baseReader
	rcUDPReader *rcudp.UDPReader
}

// NewRaptorCodeReader makes an instance of RaptorCodeReader.
func NewRaptorCodeReader(lpeerInfo encoding.PeerInfo, publisher eventbus.Publisher,
	gossip *protocol.Gossip, processor *peer.MessageProcessor) *RaptorCodeReader {
	// TODO: handle this by configs
	lpeerInfo.Port += 10000
	addr := lpeerInfo.Address()

	lAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		log.Panicf("invalid kadcast peer address %s", addr)
	}

	r := new(RaptorCodeReader)
	r.base = newBaseReader(lpeerInfo, publisher, gossip, processor)

	r.rcUDPReader, err = rcudp.NewUDPReader(lAddr, rcudp.MessageCollector(r.base.handleBroadcast))
	if err != nil {
		panic(err)
	}

	log.WithField("l_addr", lAddr.String()).Infoln("Starting Reader")
	return r
}

// Close closes reader TCP listener.
func (r *RaptorCodeReader) Close() error {
	if r.rcUDPReader != nil {
		// TODO: r.rcUDPReader.Close()
	}

	return nil
}

// Serve starts accepting and processing TCP connection and packets.
func (r *RaptorCodeReader) Serve() {
	r.rcUDPReader.Serve()
}

func rcudpWrite(laddr, raddr net.UDPAddr, blocks [][]byte) {
	raddr.Port += 10000

	if err := rcudp.WriteBlocks(&laddr, &raddr, blocks); err != nil {
		log.WithError(err).
			WithField("dest", raddr.String()).
			Error("rcudp write packet failed")
	}
}
