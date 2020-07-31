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
	haltChan chan<- HaltMsg
	lock     sync.RWMutex
	t        *time.Timer
}

// NewTimer instantiates a new Timer
func NewTimer(haltChan chan<- HaltMsg) *Timer {
	return &Timer{
		haltChan: haltChan,
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
	t.haltChan <- HaltMsg{
		Hash: emptyHash[:],
		Sv:   []*message.StepVotes{},
	}
}
