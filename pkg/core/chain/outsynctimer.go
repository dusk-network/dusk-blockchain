// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type outSyncTimer struct {
	// Timeout duration before executing timer callback
	timeout time.Duration
	// callback to be executed when timer expires
	callback func() error

	lock sync.Mutex
	// ownerID is the ID (IPv4) addr of the peer that initiated outSync mode
	ownerID    string
	cancelChan chan bool
	t          *time.Timer
}

func newSyncTimer(timeout time.Duration, onExpiredFn func() error) *outSyncTimer {
	return &outSyncTimer{
		timeout:  timeout,
		callback: onExpiredFn,
	}
}

// Start starts the syncTimer after ensuring it's not running.
// ownerID should be the address of the peer initiating a sync procedure.
func (s *outSyncTimer) Start(id string) {
	s.Cancel()

	// initialize new timer event Consumer
	s.lock.Lock()
	s.ownerID = id
	s.cancelChan = make(chan bool, 1)
	s.t = time.NewTimer(s.timeout)
	eventChan := s.t.C
	s.lock.Unlock()

	go eventConsumer(eventChan, s.cancelChan, s.callback, id)
}

// Cancel terminates timer and cancel eventConsumer goroutine, if exists.
func (s *outSyncTimer) Cancel() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.t != nil {
		s.t.Stop()
		s.t = nil

		// Cancel eventConsumer goroutine
		s.cancelChan <- true
	}
}

// Reset re-calculate and reset timer expiry timestamp.
func (s *outSyncTimer) Reset(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.t == nil {
		// No timer started
		return nil
	}

	if s.ownerID != id {
		// outSyncTimer can be reset only by the ID that has started the timer
		return errors.New("wrong ownership")
	}

	s.t.Reset(s.timeout)

	return nil
}

// eventConsumer is statless consumer of time.Timer event.
func eventConsumer(event <-chan time.Time, cancelChan chan bool, onExpiredFn func() error, id string) {
	select {
	case <-cancelChan:
		return
	case <-event:
		// TODO: Increase ban score for the dishonest Peer
		// Trigger callback
		logrus.WithField("dishonest_peer", id).Warn("synchronizer timer triggered")

		if err := onExpiredFn(); err != nil {
			logrus.WithError(err).Warn("outsynctimer expiry callback err")
		}
	}
}
