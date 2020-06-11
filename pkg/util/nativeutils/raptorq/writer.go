// +build

package raptorq

import (
	"bytes"
	"net"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/harmony-one/go-raptorq/pkg/defaults"
	"github.com/harmony-one/go-raptorq/pkg/raptorq"
	log "github.com/sirupsen/logrus"
)

// encodeWrite runs RaptorQ encoder, writes encoding symbols multiple times to the conn
func encodeWrite(conn *net.UDPConn, sourceObject []byte, overhead uint8) error {

	// Configure algorithm
	var err error
	var enc raptorq.Encoder

	digest, err := hash.Xxhash(sourceObject)
	if err != nil {
		return err
	}

	enc, err = defaults.NewEncoder(sourceObject, SymbolSize, uint16(SymbolSize), MaxSubBlockSize, 4)
	if err != nil {
		return err
	}

	commonOTI := enc.CommonOTI()
	schemeSpecOTI := enc.SchemeSpecificOTI()

	packets := make([][]byte, 0)

	sbnCount := enc.NumSourceBlocks()
	for sbn := uint8(0); sbn < sbnCount; sbn++ {

		numEsi := enc.MinSymbols(sbn)
		for esi := uint16(0); esi < numEsi+uint16(overhead); esi++ {

			symbol := make([]byte, SymbolSize)
			var written uint
			written, err = enc.Encode(sbn, uint32(esi), symbol)
			if err != nil {
				break
			}

			p := NewPacket(digest, sbn, esi, symbol[0:written], commonOTI, schemeSpecOTI)

			var buf bytes.Buffer
			if err = p.MarshalBinary(&buf); err != nil {
				break
			}

			if _, err = conn.Write(buf.Bytes()); err != nil {
				break
			}

			packets = append(packets, buf.Bytes())
		}

		if err != nil {
			break
		}
	}

	enc.Close()

	if err != nil {
		log.WithError(err).Tracef("raptorq encoder failed")
		return err
	}

	log.Tracef("NumSourceBlocks %d, overall symbols %d, commonOTI %d, schemeSpecOTI %d", sbnCount, len(packets), commonOTI, schemeSpecOTI)
	return nil
}

func SendUDP(laddr, raddr *net.UDPAddr, sourceObject []byte, overhead uint8) error {

	var err error
	var conn *net.UDPConn

	// Send from same IP that the UDP listener is bound on but choose random port
	laddr.Port = 0
	conn, err = net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return err
	}

	err = encodeWrite(conn, sourceObject, overhead)
	_ = conn.Close()

	return err
}
