package rcudp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net"
	"sync"
	"time"

	fountain "github.com/google/gofountain"
	logger "github.com/sirupsen/logrus"
)

var log = logger.WithFields(logger.Fields{"process": "rcudp"})

// msgID alias
type msgID [8]byte

type message struct {
	decoder   *Decoder
	srcAddr   net.UDPAddr
	collected bool
	recv_time int64
}

// MessageCollector callback to be run on a newly decoded message
type MessageCollector func(addr string, decoded []byte) error

// UDPReader that supports decoding Raptor codes packets
type UDPReader struct {
	lAddr *net.UDPAddr

	lock    sync.RWMutex
	objects map[msgID]*message

	collector MessageCollector
}

// NewUDPReader instantiate a UDP reader of raptor code packets
func NewUDPReader(lAddr *net.UDPAddr, h MessageCollector) (*UDPReader, error) {
	return &UDPReader{
		objects:   make(map[msgID]*message),
		lAddr:     lAddr,
		collector: h,
	}, nil
}

// Serve reads data from UDP socket and tries to re-assemble the sourceObject
func (r *UDPReader) Serve() {

	listener, err := net.ListenUDP("udp4", r.lAddr)
	if err != nil {
		log.Panic(err)
	}

	log.WithField("addr", r.lAddr.String()).
		Infof("Start Raptor code UDPReader")

	go r.cleanup()

	for {
		b := make([]byte, maxPacketLen)
		n, uAddr, err := listener.ReadFromUDP(b)
		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			continue
		}

		go func() {
			r.lock.Lock()
			if err := r.processPacket(*uAddr, b[:n]); err != nil {
				log.WithError(err).Warn("Error on packet processing")
			}
			r.lock.Unlock()
		}()
	}
}

func (r *UDPReader) processPacket(srcAddr net.UDPAddr, data []byte) error {

	p := Packet{}
	buf := bytes.NewBuffer(data)
	if err := p.unmarshalBinary(buf); err != nil {
		return err
	}

	//log.WithField("receiver", r.lAddr.Port).
	//	Infof("Received packet:  oID %s, bID %d, TL %d, NSS %d ",
	//		hex.EncodeToString(p.objectID[:]), p.blockID, p.transferLength, p.NumSourceSymbols)

	var m *message
	var ok bool
	if m, ok = r.objects[p.messageID]; !ok {
		// Instantiate a new decoder for handling the packet
		// a decoder per packet
		d := NewDecoder(int(p.NumSourceSymbols),
			symbolAlignmentSize, int(p.transferLength),
			int(p.PaddingSize))

		m = &message{
			decoder:   d,
			srcAddr:   srcAddr,
			recv_time: time.Now().Unix(),
			collected: false,
		}

		r.objects[p.messageID] = m
	}

	if m == nil {
		err := errors.New("nil message")
		log.WithError(err).Warn("UDP Reader failed")
		return err
	}

	if m.collected {
		// this message has been already decoded and collected
		return nil
	}

	// Ensure the source address of this encoding symbol is the same as the primary one
	if !addrEqual(m.srcAddr, srcAddr) {
		err := errors.New("encoding symbols of same source object cannot be from different UDP addresses")
		log.WithError(err).Warn("UDP Reader failed")
		return err
	}

	b := fountain.LTBlock{
		BlockCode: int64(p.blockID),
		Data:      p.block[:],
	}

	decoded := m.decoder.AddBlock(b)
	if decoded != nil {
		// run callback here
		m.collected = true
		err := r.collector(srcAddr.String(), decoded)
		if err != nil {
			return err
		}

		// At that point in time, the object(message) is already decoded and
		// collected. However, we can not delete it immediately. This is because
		// more blocks of this message will probably arrive in the next second
		// or two. Here the staleTimeout plays its role
	}
	return nil
}

// cleanup checks for stale and consumed messages. If found, deletes them
func (r *UDPReader) cleanup() {

	for {

		time.Sleep(time.Duration(staleTimeout) * time.Second)

		deletionList := make([][8]byte, 0)
		r.lock.RLock()
		for k, v := range r.objects {

			// message not consumed and staleTimeout has been reached
			if (time.Now().Unix() - v.recv_time) > staleTimeout {
				deletionList = append(deletionList, k)

				// this message is out of time. Pending to be deleted. if not
				// collected yet, that might mean staleTimeout should be
				// increased or message delivery simply failed
				if !v.collected {
					d := v.decoder
					log.WithField("receiver", r.lAddr.Port).
						Warnf("Not collected message with oID %s, NumSourceSymbols %d, PaddingSize %d",
							hex.EncodeToString(k[:]), d.numSourceSymbols, d.paddingSize)
				}
			}
		}
		r.lock.RUnlock()

		if len(deletionList) == 0 {
			continue
		}

		// delete items
		r.lock.Lock()
		for _, key := range deletionList {
			delete(r.objects, key)
		}
		r.lock.Unlock()
	}

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
