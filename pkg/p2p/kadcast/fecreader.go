// +build

package kadcast

import (
	"bytes"
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/raptorq"
)

// FECReader is Forward Error Correction reader based on UDP transport to transmit broadcast packets
// This is an alternative of TCP Reader based on RaptorQ
type FECReader struct {
	publisher eventbus.Publisher
	gossip    *protocol.Gossip

	// lpeer is the tuple identifying this peer
	lpeer encoding.PeerInfo

	raptorqReader *raptorq.UDPReader
	messageRouter messageRouter
}

// FECReader makes a new kadcast reader that handles TCP packets of broadcasting
func NewFECReader(lpeerInfo encoding.PeerInfo, publisher eventbus.Publisher, gossip *protocol.Gossip, dupeMap *dupemap.DupeMap) *FECReader {

	addr := lpeerInfo.Address()
	lAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		log.Panicf("invalid kadcast peer address %s", addr)
	}

	lAddr.Port += 10

	r, err := raptorq.NewUDPReader(lAddr)
	if err != nil {
		panic(r)
	}

	reader := &FECReader{
		raptorqReader: r,
		lpeer:         lpeerInfo,
		messageRouter: messageRouter{publisher: publisher, dupeMap: dupeMap},
		publisher:     publisher,
		gossip:        gossip,
	}

	log.WithField("l_addr", lAddr.String()).Infoln("Starting Reader")

	return reader
}

// Close closes reader TCP listener
func (r *FECReader) Close() error {
	if r.raptorqReader != nil {
		// TODO: r.raptorqReader.Close()
	}
	return nil
}

// Serve starts accepting and processing TCP connection and packets
func (r *FECReader) Serve() {

	go r.raptorqReader.Serve()

	for {
		id := <-r.raptorqReader.ReadySourceObjectChan
		decodedObject, uAddr, err := r.raptorqReader.CollectDecodedSourceObject(id)
		if err != nil {
			log.WithError(err).Warnf("collecting decoded source object failed")
			continue
		}

		r.handleBroadcast(uAddr.String(), decodedObject)
	}
}

func (r *FECReader) handleBroadcast(raddr string, b []byte) error {

	// Unmarshal message header
	buf := bytes.NewBuffer(b)
	var header encoding.Header
	if err := header.UnmarshalBinary(buf); err != nil {
		log.WithError(err).Warn("TCP reader rejects a packet")
		return err
	}

	// Run extra checks over message data
	if err := isValidMessage(raddr, header); err != nil {
		log.WithError(err).Warn("TCP reader rejects a packet")
		return err
	}

	// Unmarshal broadcast message payload
	var p encoding.BroadcastPayload
	if err := p.UnmarshalBinary(buf); err != nil {
		log.WithError(err).Warn("could not unmarshal message")
		return err
	}

	// Read `message` from gossip frame
	buf = bytes.NewBuffer(p.GossipFrame)
	message, err := r.gossip.ReadFrame(buf)
	if err != nil {
		log.WithError(err).Warnln("could not read the gossip frame")
		return err
	}

	// Propagate message to the node router respectively eventbus
	// Non-routable and duplicated messages are not repropagated
	err = r.messageRouter.Collect(message, p.Height)
	if err != nil {
		log.WithError(err).Errorln("error routing message")
		return err
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

func rqSendUDP(laddr, raddr net.UDPAddr, payload []byte) {

	raddr.Port += 10

	log.WithField("dest", raddr.String()).Tracef("Sending raptorq udp packet of len %d", len(payload))
	if err := raptorq.SendUDP(&laddr, &raddr, payload, 5); err != nil {
		log.WithError(err).WithField("dest", raddr.String()).Warnf("Sending raptorq udp packet of len %d failed", len(payload))
	}
}
