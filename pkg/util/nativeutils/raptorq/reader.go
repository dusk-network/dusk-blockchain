// +build

package raptorq

import (
	"bytes"
	"errors"
	"net"
	"sync"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/harmony-one/go-raptorq/pkg/defaults"
	"github.com/harmony-one/go-raptorq/pkg/raptorq"

	log "github.com/sirupsen/logrus"
)

type sourceObject struct {
	decoder raptorq.Decoder
	id      SourceObjectID
	srcAddr net.UDPAddr
}

// UDPReader that supports decoding RaptorQ packets
type UDPReader struct {
	lAddr *net.UDPAddr

	lock    sync.RWMutex
	objects map[SourceObjectID]*sourceObject

	// TODO:Depends on swig.WaitForSourceObject impl
	ReadySourceObjectChan chan [8]byte
}

func NewUDPReader(lAddr *net.UDPAddr) (*UDPReader, error) {
	return &UDPReader{
		objects:               make(map[SourceObjectID]*sourceObject),
		lAddr:                 lAddr,
		ReadySourceObjectChan: make(chan [8]byte, 1000)}, nil
}

// Serve reads data from UDP socket and tries to re-assemble the sourceObject (RaptorQ)
func (r *UDPReader) Serve() {

	listener, err := net.ListenUDP("udp4", r.lAddr)
	if err != nil {
		log.Panic(err)
	}

	for {
		b := make([]byte, MaxUDPLength)
		n, uAddr, err := listener.ReadFromUDP(b)
		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			continue
		}

		if err := r.processPacket(*uAddr, b[0:n]); err != nil {
			log.WithError(err).Warn("Error on packet processing")
		}
	}
}

func (r *UDPReader) processPacket(srcAddr net.UDPAddr, data []byte) error {

	p := Packet{}
	buf := bytes.NewBuffer(data)
	if err := p.UnmarshalBinary(buf); err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	var s *sourceObject
	var ok bool
	if s, ok = r.objects[p.sourceObjectID]; !ok {
		// Instantiate a new decoder for handling the packet
		// a decoder per packet
		d, err := defaults.NewDecoder(p.commonOTI, p.schemeSpecOTI, [8]byte(p.sourceObjectID))
		if err != nil {
			return err
		}

		d.AddReadySourceObjectChan(r.ReadySourceObjectChan)
		s = &sourceObject{
			decoder: d,
			id:      p.sourceObjectID,
			srcAddr: srcAddr,
		}

		r.objects[p.sourceObjectID] = s
	}

	// Ensure the source address of this encoding symbol is the same as the primary one
	if !addrEqual(s.srcAddr, srcAddr) {
		err := errors.New("encoding symbols of same source object cannot be from different UDP addresses")
		log.WithError(err).Warn("UDP Reader failed")
		return err
	}

	if s != nil {
		s.decoder.Decode(p.sbn, uint32(p.esi), p.symbolData[:])
	} else {
		err := errors.New("nil source object")
		log.WithError(err).Warn("UDP Reader failed")
		return err
	}

	return nil
}

func addrEqual(a1, a2 net.UDPAddr) bool {

	if !a1.IP.Equal(a2.IP) {
		return false
	}

	if a1.Port != a2.Port {
		return false
	}

	return true
}

// CollectDecodedSourceObject makes an attempt to collect complete source object
// If succeeds, it also releases the object
func (r *UDPReader) CollectDecodedSourceObject(id [8]byte) ([]byte, *net.UDPAddr, error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	object, ok := r.objects[id]
	if !ok {
		return nil, nil, errors.New("not found")
	}

	addr := &object.srcAddr

	transferLen := object.decoder.TransferLength()
	decodedObject := make([]byte, transferLen)

	var err error
	if _, err = object.decoder.SourceObject(decodedObject); err != nil {
		return nil, addr, err
	}

	// Run Checksum
	digest, err := hash.Xxhash(decodedObject)
	if err != nil {
		return nil, addr, err
	}

	if !bytes.Equal(digest[:8], id[:]) {
		return nil, addr, errors.New("invalid checksum")
	}

	// Release object
	delete(r.objects, id)

	return decodedObject, addr, nil
}
