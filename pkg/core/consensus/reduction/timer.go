package reduction

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
)

type Timer struct {
	requestHalt func([]byte, ...*agreement.StepVotes)
	t           *time.Timer
}

func NewTimer(requestHalt func([]byte, ...*agreement.StepVotes)) *Timer {
	return &Timer{
		requestHalt: requestHalt,
	}
}

func (t *Timer) Start(timeOut time.Duration) {
	t.t = time.NewTimer(timeOut)
}

func (t *Timer) Stop() {
	t.t.Stop()
}

func (t *Timer) Trigger() {
	t.requestHalt(nil)
}
