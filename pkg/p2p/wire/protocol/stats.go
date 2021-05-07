// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

// nolint
package protocol

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var s stats

type stats struct {
	// cumulativeDuration cumulative arrival time
	cumulativeDuration int64

	// messageCounter number of messages registered
	messageCounter int64

	// maxPacketLength recently registered
	maxPacketLength uint64

	lock sync.Mutex
}

// registerPacket reports stats based on collected data of the last 1000 messages.
// this can be enabled manually in case of evaluating network performance.
func (s *stats) registerPacket(packetLength uint64, timestamp int64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cumulativeDuration += (time.Now().UnixNano() - timestamp) / 1000000
	s.messageCounter++

	if packetLength > s.maxPacketLength {
		s.maxPacketLength = packetLength
	}

	if s.messageCounter%1000 == 0 {
		var averageArrivalTime int64
		if s.messageCounter > 0 {
			averageArrivalTime = s.cumulativeDuration / s.messageCounter
		}

		logrus.WithField("cumulativeDuration", s.cumulativeDuration).WithField("maxPacketLength", s.maxPacketLength).
			WithField("messages", s.messageCounter).WithField("average_ms", averageArrivalTime).Info("Gossip Stats")

		// Reset counters
		s.messageCounter = 0
		s.cumulativeDuration = 0
		s.maxPacketLength = 0
	}
}
