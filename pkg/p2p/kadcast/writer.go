// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rcudp"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

const (
	// MaxWriterQueueSize max number of messages queued for broadcasting.
	// While in Gossip there is a Writer per a peer, in Kadcast there a single Writer.
	// That's why it should be higher than queue size in Gossip.
	MaxWriterQueueSize = 10000
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

	// Low Priority message rate limiter
	limiter *rate.Limiter
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
		limiter:           rate.NewLimiter(rate.Every(time.Second/60), 1),
	}
}

// ServeOverChan processes any kadcast messaging to the wire.
//nolint
func (w *Writer) ServeOverChan() {
	// NewChanListener is preferred here as it passes message.Message to the
	// Write, where NewStreamListener works with bytes.Buffer only.
	// Later this could be change if perf issue noticed.
	writeQueue := make(chan message.Message, MaxWriterQueueSize)
	w.kadcastSubscription = w.subscriber.Subscribe(topics.Kadcast, eventbus.NewChanListener(writeQueue))

	writePointMsgQueue := make(chan message.Message, MaxWriterQueueSize)
	w.kadcastPointSubscription = w.subscriber.Subscribe(topics.KadcastPoint, eventbus.NewChanListener(writePointMsgQueue))

	go func() {
		for msg := range writeQueue {
			go func(m message.Message) {
				buf := m.Payload().(message.SafeBuffer)
				if err := w.WriteToAll(buf.Bytes(), m.Header(), 0); err != nil {
					log.WithError(err).Warn("kadcast write failed")
				}
			}(msg)
		}
	}()

	go func() {
		for msg := range writePointMsgQueue {
			go func(m message.Message) {
				buf := m.Payload().(message.SafeBuffer)
				if err := w.WriteToPoint(buf.Bytes(), m.Header(), 0); err != nil {
					log.WithError(err).Warn("kadcast-point write failed")
				}
			}(msg)
		}
	}()
}

// Serve is based on StreamListener while ServeOverChan is based on ChanListener.
func (w *Writer) Serve() {
	priorityMapper := func(t topics.Topic) byte {
		switch t {
		case topics.Tx:
			return 0
		case topics.Block:
			return 2
		case topics.NewBlock:
			return 3
		case topics.Reduction:
			return 4
		case topics.Agreement:
			return 5
		case topics.Candidate, topics.GetCandidate:
			return 6
		}

		return 1
	}

	l1 := eventbus.NewStreamListenerWithParams(w, MaxWriterQueueSize, priorityMapper)
	w.kadcastSubscription = w.subscriber.Subscribe(topics.Kadcast, l1)

	l2 := eventbus.NewStreamListenerWithParams(w, MaxWriterQueueSize, priorityMapper)
	w.kadcastPointSubscription = w.subscriber.Subscribe(topics.KadcastPoint, l2)
}

func (w *Writer) Write(data, header []byte, priority byte) (int, error) {
	go func() {
		// Rate limiter for 0-priority messages
		if priority == 0 {
			now := time.Now()
			rv := w.limiter.ReserveN(now, 1)
			delay := rv.DelayFrom(now)
			time.Sleep(delay)
		}

		var err error

		if len(header) > 1 {
			err = w.WriteToPoint(data, header, priority)
		}

		if len(header) == 1 {
			err = w.WriteToAll(data, header, priority)
		}

		if err != nil {
			log.WithError(err).Warn("write failed")
		}
	}()

	return 0, nil
}

// WriteToAll broadcasts message to the entire network.
func (w *Writer) WriteToAll(data, header []byte, priority byte) error {
	if len(header) == 0 {
		return errors.New("empty message header")
	}

	// Constuct gossip frame.
	buf := bytes.NewBuffer(data)
	if err := w.gossip.Process(buf); err != nil {
		return err
	}

	return w.broadcastPacket(header[0], buf.Bytes())
}

// WriteToPoint writes a message to a single destination.
// The receiver address is read from message Header.
func (w *Writer) WriteToPoint(data, header []byte, priority byte) error {
	// Height = 0 disables re-broadcast algorithm in the receiver node. That
	// said, sending a message to peer with height 0 will be received by the
	// destination peer but will not be repropagated to any other node.
	const height = byte(0)

	if len(header) == 0 {
		return errors.New("empty message header")
	}

	raddr := string(header)

	delegates := make([]encoding.PeerInfo, 1)

	var err error

	delegates[0], err = encoding.MakePeerFromAddr(raddr)
	if err != nil {
		return err
	}

	// Constuct gossip frame.
	buf := bytes.NewBuffer(data)
	if err = w.gossip.Process(buf); err != nil {
		return err
	}

	// Marshal message data
	var packet []byte

	packet, err = w.marshalBroadcastPacket(height, buf.Bytes())
	if err != nil {
		return err
	}

	var blocks [][]byte

	if w.raptorCodeEnabled {
		var err error

		// Compile blocks only once but send them to multiple delegates
		_, blocks, err = rcudp.CompileRaptorRFC5053(0, packet, redundancyFactor)
		if err != nil {
			return err
		}
	}

	// Send message to a single destination using height = 0.
	return w.sendToDelegates(delegates, height, blocks, packet)
}

// BroadcastPacket sends a `CHUNKS` message across the network
// following the Kadcast broadcasting rules with the specified height.
func (w *Writer) broadcastPacket(maxHeight byte, payload []byte) error {
	if maxHeight == 0 {
		// last subtree, no more broadcast needed
		return nil
	}

	if maxHeight > byte(len(w.router.tree.buckets)) {
		return fmt.Errorf("invalid max height %d, %d", maxHeight, len(w.router.tree.buckets))
	}

	if log.Logger.GetLevel() == logrus.TraceLevel {
		log.WithField("l_addr", w.router.LpeerInfo.String()).WithField("max_height", maxHeight).
			Traceln("broadcasting procedure")
	}

	// Marshal message data
	packet, err := w.marshalBroadcastPacket(0, payload)
	if err != nil {
		return err
	}

	var blocks [][]byte

	if w.raptorCodeEnabled {
		var err error

		// RaptroRFC5053 algorithm is performed only once to compile raptor
		// blocks for a single message. Broadcast Height field is modified
		// accordingly depending on destination sub-tree
		_, blocks, err = rcudp.CompileRaptorRFC5053(0, packet, redundancyFactor)
		if err != nil {
			return err
		}
	}

	for h := byte(0); h <= maxHeight-1; h++ {
		// Fetch delegating nodes based on height value
		delegates := w.fetchDelegates(h)
		if len(delegates) == 0 {
			continue
		}

		// Send to all delegates
		if err := w.sendToDelegates(delegates, h, blocks, packet); err != nil {
			log.WithError(err).Warnln("send to delegates failed")
		}
	}

	return nil
}

func (w *Writer) marshalBroadcastPacket(h byte, payload []byte) ([]byte, error) {
	encHeader := makeHeader(encoding.BroadcastMsg, w.router)

	p := encoding.BroadcastPayload{
		Height:      h,
		GossipFrame: payload,
	}

	var buf bytes.Buffer
	if err := encoding.MarshalBinary(encHeader, &p, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (w *Writer) fetchDelegates(h byte) []encoding.PeerInfo {
	router := w.router
	myPeer := w.router.LpeerInfo

	// this should be always a deep copy of a bucket from the tree
	var b bucket

	router.tree.mu.RLock()
	b = router.tree.buckets[h]
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

		err := getRandDelegates(router.beta, in, &delegates)
		if err != nil {
			log.WithError(err).Warn("get rand delegates failed")
		}
	}

	return delegates
}

func (w *Writer) sendToDelegates(delegates []encoding.PeerInfo, height byte, blocks [][]byte, packet []byte) error {
	if len(delegates) == 0 {
		return errors.New("empty delegates list")
	}

	var failureRate int
	// For each of the delegates found from this bucket, make an attempt to
	// repropagate Broadcast message
	for _, destPeer := range delegates {
		if w.router.LpeerInfo.IsEqual(destPeer) {
			log.Warn("dest delegate same as sender")
			failureRate++
			continue
		}

		if log.Logger.GetLevel() == logrus.TraceLevel {
			// Avoid wasting CPU cycles for WithField construction in non-trace level
			log.WithField("l_addr", w.router.LpeerInfo.String()).
				WithField("r_addr", destPeer.String()).
				WithField("height", height).
				WithField("raptor", w.raptorCodeEnabled).
				Trace("sending message")
		}

		// Send message to the dest peer with rc-udp or tcp
		if w.raptorCodeEnabled {
			laddr := w.router.lpeerUDPAddr
			raddr := destPeer.GetUDPAddr()
			raddr.Port += 10000

			// Write all raptor blocks
			// Failing to send message to a single delegate is not critical.
			if err := rcudp.WriteBlocks(&laddr, &raddr, blocks, height); err != nil {
				failureRate++

				log.WithError(err).
					WithField("dest", raddr.String()).
					WithField("rate", failureRate).
					Warnln("rcudp write failed")
			}
		} else {
			tcpSend(destPeer.GetUDPAddr(), packet)
		}
	}

	if failureRate == len(delegates) {
		return fmt.Errorf("message sending failed for %d delegate(s)", len(delegates))
	}

	return nil
}

// Close unsubscribes from eventbus events.
func (w *Writer) Close() error {
	w.subscriber.Unsubscribe(topics.Kadcast, w.kadcastSubscription)
	w.subscriber.Unsubscribe(topics.KadcastPoint, w.kadcastPointSubscription)
	return nil
}
