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

// CompileRaptorRFC5053 compiles raptorRFC5053 blocks from message with a
// specified redundancyFactor.
// In Kadcast, one could compile blocks once but send them to multiple delegates.
func CompileRaptorRFC5053(height byte, message []byte, redundancyFactor uint8) ([]byte, [][]byte, error) {
	msgID, err := hash.Xxhash(message)
	if err != nil {
		return nil, nil, err
	}

	w, err := NewEncoder(message, BlockSize, redundancyFactor, symbolAlignmentSize)
	if err != nil {
		return nil, nil, err
	}

	fountainBlocks := w.GenerateBlocks()
	blocks := make([][]byte, 0, len(fountainBlocks))

	for _, b := range fountainBlocks {
		p := newPacket(
			msgID, uint16(w.NumSourceSymbols),
			w.PaddingSize, uint32(w.TransferLength()),
			uint32(b.BlockCode), b.Data, height)

		var blob []byte

		if blob, err = p.marshal(); err != nil {
			log.WithError(err).Warnf("Error writing to UDP socket")
			continue
		}

		// append to list of all blocks
		blocks = append(blocks, blob)
	}

	return msgID, blocks, nil
}

// WriteBlocks writes already compiled raptor blocks to raddr via UDP.
// It utilizes a simple back-off.
func WriteBlocks(laddr, raddr *net.UDPAddr, blocks [][]byte, height byte) error {
	// Send from same IP that the UDP listener is bound on but choose random port
	laddr.Port = 0

	conn, err := net.DialUDP("udp4", laddr, raddr)
	if err != nil {
		return err
	}

	if err = conn.SetWriteBuffer(writeBufferSize); err != nil {
		log.WithError(err).Traceln("SetWriteBuffer socket problem")
	}

	for _, blk := range blocks {
		// Update height field accordingly
		blk[BcastHeightPos] = height

		time.Sleep(backoffTimeout)

		if _, err = conn.Write(blk); err != nil {
			log.WithError(err).Warn("error writing to UDP socket")
		}
	}

	_ = conn.Close()

	return err
}
