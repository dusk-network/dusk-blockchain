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
	subscriber     eventbus.Subscriber
	subscriptionID uint32
	gossip         *protocol.Gossip

	// Kademlia routing state
	router *RoutingTable

	writeQueue     chan message.Message
	raptorqEnabled bool
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(router *RoutingTable, subscriber eventbus.Subscriber, gossip *protocol.Gossip, raptorqEnabled bool) *Writer {

	pw := &Writer{
		subscriber:     subscriber,
		router:         router,
		gossip:         gossip,
		raptorqEnabled: raptorqEnabled,
	}

	return pw
}

// Serve processes any kadcast messaging to the wire
func (w *Writer) Serve() {

	// NewChanListener is preferred here as it passes message.Message to the
	// Write, where NewStreamListener works with bytes.Buffer only.
	// Later this could be change if perf issue noticed

	w.writeQueue = make(chan message.Message, 1000)
	w.subscriptionID = w.subscriber.Subscribe(topics.Kadcast, eventbus.NewChanListener(w.writeQueue))

	go func() {
		for msg := range w.writeQueue {
			if err := w.Write(msg); err != nil {
				log.WithError(err).Trace("kadcast writer problem")
			}
		}
	}()
}

// Write expects the actual payload in a marshaled form
func (w *Writer) Write(m message.Message) error {

	header := m.Header()
	buf := m.Payload().(bytes.Buffer)

	if len(header) == 0 {
		return errors.New("invalid message height")
	}

	// Constuct gossip frame
	if err := w.gossip.Process(&buf); err != nil {
		log.WithError(err).Error("reading gossip frame failed")
		return err
	}

	w.broadcastPacket(header[0], buf.Bytes())
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
			Traceln("Sending Broadcast message")

		if w.raptorqEnabled {
			// go rqSendUDP(w.router.lpeerUDPAddr, destPeer.GetUDPAddr(), buf.Bytes())
		} else {
			go tcpSend(destPeer.GetUDPAddr(), buf.Bytes())
		}
	}
}

// Close unsubscribes from eventbus events
func (w *Writer) Close() error {
	w.subscriber.Unsubscribe(topics.Kadcast, w.subscriptionID)
	return nil
}
