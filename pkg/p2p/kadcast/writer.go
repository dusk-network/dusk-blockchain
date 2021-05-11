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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	subscriber eventbus.Subscriber
	gossip     *protocol.Gossip
	// Kademlia routing state
	router            *RoutingTable
	raptorCodeEnabled bool

	kadcastSubscription, kadcastPointSubscription uint32
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(router *RoutingTable, subscriber eventbus.Subscriber, gossip *protocol.Gossip, raptorCodeEnabled bool) *Writer {
	return &Writer{
		subscriber:        subscriber,
		router:            router,
		gossip:            gossip,
		raptorCodeEnabled: raptorCodeEnabled,
	}
}

// Serve processes any kadcast messaging to the wire.
func (w *Writer) Serve() {
	// NewChanListener is preferred here as it passes message.Message to the
	// Write, where NewStreamListener works with bytes.Buffer only.
	// Later this could be change if perf issue noticed.
	writeQueue := make(chan message.Message, 1000)
	w.kadcastSubscription = w.subscriber.Subscribe(topics.Kadcast, eventbus.NewChanListener(writeQueue))

	go func() {
		for msg := range writeQueue {
			if err := w.Write(msg); err != nil {
				log.WithError(err).Warn("kadcast writer problem")
			}
		}
	}()

	writePointMsgQueue := make(chan message.Message, 1000)
	w.kadcastPointSubscription = w.subscriber.Subscribe(topics.KadcastPoint, eventbus.NewChanListener(writePointMsgQueue))

	go func() {
		for msg := range writePointMsgQueue {
			if err := w.WriteToPoint(msg); err != nil {
				log.WithError(err).Warn("kadcast-point writer problem")
			}
		}
	}()
}

// Write expects the actual payload in a marshaled form.
func (w *Writer) Write(m message.Message) error {
	header := m.Header()
	buf := m.Payload().(message.SafeBuffer)

	if len(header) == 0 {
		return errors.New("invalid message height")
	}

	// Constuct gossip frame.
	if err := w.gossip.Process(&buf.Buffer); err != nil {
		log.WithError(err).Error("reading gossip frame failed")
		return err
	}

	w.broadcastPacket(header[0], buf.Bytes())

	return nil
}

// WriteToPoint writes a message to a single destination.
// The receiver address is read from message Header.
func (w *Writer) WriteToPoint(m message.Message) error {
	h := m.Header()
	if len(h) == 0 {
		return errors.New("empty header")
	}

	raddr := string(h)

	delegates := make([]encoding.PeerInfo, 1)

	var err error

	delegates[0], err = encoding.MakePeerFromAddr(raddr)
	if err != nil {
		return err
	}

	// Constuct gossip frame.
	buf := m.Payload().(message.SafeBuffer)
	if err := w.gossip.Process(&buf.Buffer); err != nil {
		log.WithError(err).Error("reading gossip frame failed")
		return err
	}

	// Send message to a single destination using height = 0.
	// Height = 0 disables re-broadcast algorithm in the receiver node.
	w.sendToDelegates(delegates, 0, buf.Bytes())
	return nil
}

// BroadcastPacket sends a `CHUNKS` message across the network
// following the Kadcast broadcasting rules with the specified height.
func (w *Writer) broadcastPacket(height byte, payload []byte) {
	router := w.router
	myPeer := router.LpeerInfo

	if height > byte(len(router.tree.buckets)) || height == 0 {
		return
	}

	log.WithField("l_addr", myPeer.String()).WithField("height", height).
		Traceln("Run broadcasting algorithm")

	var h byte
	for h = 0; h <= height-1; h++ {
		// Fetch delegating nodes based on height value
		delegates := w.fetchDelegates(h)

		// Send to all delegates the payload
		//
		// TODO: Optional here could be to marshal binary once but tamper height field in
		// the blob on each height change
		w.sendToDelegates(delegates, h, payload)
	}
}

func (w *Writer) fetchDelegates(H byte) []encoding.PeerInfo {
	router := w.router
	myPeer := w.router.LpeerInfo

	// this should be always a deep copy of a bucket from the tree
	var b bucket

	router.tree.mu.RLock()
	b = router.tree.buckets[H]
	router.tree.mu.RUnlock()

	if len(b.entries) == 0 {
		return nil
	}

	delegates := make([]encoding.PeerInfo, 0)

	if b.idLength == 0 {
		// the bucket B 0 only holds one specific node of distance one
		for _, p := range b.entries {
			// Find neighbor peer
			if !myPeer.IsEqual(p) {
				delegates = append(delegates, p)
				break
			}
		}
	} else {
		// As per spec:
		//	Instead of having a single delegate per bucket, we select β
		//	delegates. This severely increases the probability that at least one
		//	out of the multiple selected nodes is honest and reachable.
		in := make([]encoding.PeerInfo, len(b.entries))
		copy(in, b.entries)

		if err := generateRandomDelegates(router.beta, in, &delegates); err != nil {
			log.WithError(err).Warn("generate random delegates failed")
		}
	}

	return delegates
}

func (w *Writer) sendToDelegates(delegates []encoding.PeerInfo, H byte, data []byte) {
	if len(delegates) == 0 {
		return
	}

	localPeer := w.router.LpeerInfo

	// Marshal message data
	h := makeHeader(encoding.BroadcastMsg, w.router)

	p := encoding.BroadcastPayload{
		Height:      H,
		GossipFrame: data,
	}

	var buf bytes.Buffer
	if err := encoding.MarshalBinary(h, &p, &buf); err != nil {
		return
	}

	// For each of the delegates found from this bucket, make an attempt to
	// repropagate Broadcast message
	for _, destPeer := range delegates {
		if localPeer.IsEqual(destPeer) {
			log.Error("Destination peer must be different from the source peer")
			continue
		}

		log.WithField("l_addr", localPeer.String()).
			WithField("r_addr", destPeer.String()).
			WithField("height", H).
			WithField("raptor", w.raptorCodeEnabled).
			WithField("len", buf.Len()).
			Trace("Sending message")

		// Send message to the dest peer with rc-udp or tcp
		if w.raptorCodeEnabled {
			// rc-udp write is destructive to the input message. If more than delegates are selected,
			// duplicate the message
			messageCopy := buf.Bytes()
			if len(delegates) > 1 {
				messageCopy = make([]byte, buf.Len())
				copy(messageCopy, buf.Bytes())
			}

			go rcudpWrite(w.router.lpeerUDPAddr, destPeer.GetUDPAddr(), messageCopy)
		} else {
			go tcpSend(destPeer.GetUDPAddr(), buf.Bytes())
		}
	}
}

// Close unsubscribes from eventbus events.
func (w *Writer) Close() error {
	w.subscriber.Unsubscribe(topics.Kadcast, w.kadcastSubscription)
	w.subscriber.Unsubscribe(topics.KadcastPoint, w.kadcastPointSubscription)
	return nil
}
