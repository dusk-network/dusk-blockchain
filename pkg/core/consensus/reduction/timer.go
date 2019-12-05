package reduction

import (
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	log "github.com/sirupsen/logrus"
)

var emptyHash [32]byte

type Timer struct {
	requestHalt func([]byte, ...*agreement.StepVotes)
	lock        sync.RWMutex
	t           *time.Timer
}

func NewTimer(requestHalt func([]byte, ...*agreement.StepVotes)) *Timer {
	return &Timer{
		requestHalt: requestHalt,
	}
}

func (t *Timer) Start(timeOut time.Duration) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.t = time.AfterFunc(timeOut, t.Trigger)
}

func (t *Timer) Stop() {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if t.t != nil {
		t.t.Stop()
	}
}

func (t *Timer) Trigger() {
	log.WithField("process", "reduction timer").Debugln("timer triggered")
	t.requestHalt(emptyHash[:])
}
