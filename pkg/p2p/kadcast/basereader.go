// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// baseReader implements the common part between both TCPReader and
// RaptorCodeReader Both readers are capable of processing Broadcast-type
// messages in kadcast but in different transport.
type baseReader struct {
	publisher eventbus.Publisher
	gossip    *protocol.Gossip
	processor *peer.MessageProcessor

	// lpeer is the tuple identifying this peer
	lpeer encoding.PeerInfo
}

func newBaseReader(lpeerInfo encoding.PeerInfo, publisher eventbus.Publisher,
	gossip *protocol.Gossip, processor *peer.MessageProcessor) *baseReader {
	return &baseReader{
		lpeer:     lpeerInfo,
		publisher: publisher,
		gossip:    gossip,
		processor: processor,
	}
}

func (r *baseReader) handleBroadcast(raddr string, b []byte) error {
	ll := log.WithField("process", "handle_broadcast")

	var header encoding.Header

	// Unmarshal message header
	buf := bytes.NewBuffer(b)
	if err := header.UnmarshalBinary(buf); err != nil {
		ll.WithError(err).Warn("reader rejects a packet")
		return err
	}

	var remotePeer *encoding.PeerInfo

	// Run extra checks over message data
	var err error
	if remotePeer, err = isValidMessage(raddr, header); err != nil {
		ll.WithError(err).Warn("reader rejects a packet")
		return err
	}

	// Unmarshal broadcast message payload
	var p encoding.BroadcastPayload
	if err = p.UnmarshalBinary(buf); err != nil {
		log.WithError(err).Warn("could not unmarshal message")
		return err
	}

	// Read `message` from gossip frame
	buf = bytes.NewBuffer(p.GossipFrame)

	m, err := r.gossip.ReadFrame(buf)
	if err != nil {
		ll.WithError(err).Warnln("could not read the gossip frame")
		return err
	}

	// Register message in the global message registry for stats collecting
	// diagnostics.RegisterWireMsg(topics.Kadcast.String(), packet)

	// TODO: set service flag properly
	respBufs, err := r.processor.Collect(raddr, m, nil, protocol.FullNode, []byte{p.Height})
	if err != nil {
		ll.WithError(err).Error("messageProcessor failed to collect message")
	}

	// Any response message is translated into Kadcast Point-to-Point wire message.
	// In other words, any bufs item is sent back to the sender node (remotePeer).
	for i := 0; i < len(respBufs); i++ {
		ll.WithField("raddr", remotePeer.Address()).Trace("send point-to-point message")

		// Send Kadcast point-to-point message with destination raddr
		r.publisher.Publish(topics.KadcastPoint,
			message.NewWithHeader(topics.KadcastPoint, respBufs[i], []byte(remotePeer.Address())))
	}

	// Repropagate message here

	// From spec:
	//	When a node receives a CHUNK, it repeats the process in a store-and-
	//	forward manner: it buffers the data, picks a random node from its
	//	buckets up to (but not including) height h, and forwards the CHUNK with
	//	a smaller value for h accordingly.

	// NB Currently, repropagate in kadcast is fully delegated to the receiving
	// component. That's needed because only the receiving component is capable
	// of verifying message fully. E.g Chain component can verifies a new block

	return nil
}

func isValidMessage(remotePeerIP string, header encoding.Header) (*encoding.PeerInfo, error) {
	// Reader handles only broadcast-type messages
	if header.MsgType != encoding.BroadcastMsg {
		return nil, errors.New("message type not supported")
	}

	// Make remote peerInfo based on addr from IP datagram and RemotePeerPort
	// from header
	remotePeer, err := encoding.MakePeerFromIP(remotePeerIP, header.RemotePeerPort)
	if err != nil {
		return nil, err
	}

	// Ensure the RemotePeerID from header is correct one
	// This together with Nonce-PoW aims at providing a bit of DDoS protection
	if !bytes.Equal(remotePeer.ID[:], header.RemotePeerID[:]) {
		return nil, errors.New("invalid remote peer id")
	}

	return &remotePeer, nil
}
