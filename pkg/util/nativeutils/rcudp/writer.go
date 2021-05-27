// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rcudp

import (
	"net"
	"time"

	"github.com/dusk-network/dusk-crypto/hash"
)

// sendRaptorRFC5053 performs raptor RFC 5053 encoding and writes to the udp socket each of the blocks.
func sendRaptorRFC5053(conn *net.UDPConn, message []byte, redundancyFactor uint8) error {
	msgID, err := hash.Xxhash(message)
	if err != nil {
		return err
	}

	w, err := NewEncoder(message, BlockSize, redundancyFactor, symbolAlignmentSize)
	if err != nil {
		return err
	}

	blocks := w.GenerateBlocks()

	for _, b := range blocks {
		p := newPacket(
			msgID, uint16(w.NumSourceSymbols),
			w.PaddingSize, uint32(w.TransferLength()),
			uint32(b.BlockCode), b.Data)

		var blob []byte

		if blob, err = p.marshal(); err != nil {
			log.WithError(err).Warnf("Error writing to UDP socket")
			continue
		}

		// Artificial delay here to avoid exceeding sender buffer size
		// Ideally, instead of a sleep here, the write could be embedded into fountain.EncodeLTBlocks.
		// This might replace the need of the artificial delay.
		time.Sleep(backoffTimeout)

		// TODO: Consider if here we can get sender_buffer error here
		if _, err = conn.Write(blob); err != nil {
			log.WithError(err).Warnf("Error writing to UDP socket")
		}
	}

	return err
}

// Write writes a message to UDP receiver in form of fountain codes.
func Write(laddr, raddr *net.UDPAddr, message []byte, redundancyFactor uint8) error {
	// Send from same IP that the UDP listener is bound on but choose random port
	laddr.Port = 0

	conn, err := net.DialUDP("udp4", laddr, raddr)
	if err != nil {
		return err
	}

	if err = conn.SetWriteBuffer(writeBufferSize); err != nil {
		log.WithError(err).Traceln("SetWriteBuffer socket problem")
	}

	err = sendRaptorRFC5053(conn, message, redundancyFactor)
	_ = conn.Close()
	return err
}
