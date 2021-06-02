// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rcudp

import (
	"encoding/binary"
	"errors"
	"time"
)

const (

	// Wire message/packet configs.

	// maxUDPLength number of bytes to transmit avoiding IP fragmentation assuming 1500 MTU.
	// NB This value must be multiple of symbolAlignmentSize.
	maxUDPLength = 1500 - 8 - 20 // − 8 byte UDP header − 20 byte IPv4 header

	// maxPacketLen is the max size of a single wire packet.
	maxPacketLen = maxUDPLength

	// Encoder configs.

	// BlockSize - max length of an encoding symbol that can fit into a single wire packet.
	BlockSize = maxUDPLength - 8 - 2 - 2 - 4 - 4 - 8

	// symbolAlignmentSize = Al is the size of each symbol in the source message in bytes.
	// Usually 4. This is the XOR granularity in bytes. On 32-byte machines 4-byte XORs.
	// will be most efficient. On the other hand, the code will perform with less overhead
	// with larger numbers of source blocks.
	symbolAlignmentSize = 4

	// UDPReader configs.

	// Messages are considered stale when more than staleTimeout seconds pass
	// after receiving the first block of the message.
	staleTimeout = int64(10)
	// UDP Recv buffer size.
	readBufferSize = 208 * 1024

	// Writer configs.
	backoffTimeout = 50 * time.Microsecond
	// UDP Sender buffer size.
	writeBufferSize = 208 * 1024
)

var (
	// byteOrder is default byte order used for the numerical fields in a packet.
	byteOrder = binary.LittleEndian

	// ErrTooLargeUDP packet cannot fit into default MTU of 1500.
	ErrTooLargeUDP = errors.New("packet cannot fit into default MTU of 1500")
)
