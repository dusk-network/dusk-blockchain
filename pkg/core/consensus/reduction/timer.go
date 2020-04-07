package reduction

import (
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	log "github.com/sirupsen/logrus"
)

var emptyHash [32]byte

// Timer sets a timeout for collecting reduction messages if no quorum is
// reached after a while
type Timer struct {
	requestHalt func([]byte, ...*message.StepVotes)
	lock        sync.RWMutex
	t           *time.Timer
}

// NewTimer instantiates a new Timer
func NewTimer(requestHalt func([]byte, ...*message.StepVotes)) *Timer {
	return &Timer{
		requestHalt: requestHalt,
	}
}

// Start the timer
func (t *Timer) Start(timeOut time.Duration) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.t = time.AfterFunc(timeOut, t.Trigger)
}

// Stop the timer
func (t *Timer) Stop() {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if t.t != nil {
		t.t.Stop()
	}
}

// Trigger the timeout and requests a halt with empty block hash
func (t *Timer) Trigger() {
	log.WithField("process", "reduction timer").Debugln("timer triggered")
	t.requestHalt(emptyHash[:])
}
