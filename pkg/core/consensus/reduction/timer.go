package reduction

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type Timer struct {
	requestHalt func([]byte, ...*agreement.StepVotes)
	t           *time.Timer
}

func NewTimer(publisher eventbus.Publisher, requestHalt func([]byte, ...*agreement.StepVotes)) *Timer {
	return &Timer{
		requestHalt: requestHalt,
	}
}

func (t *Timer) Start(timeOut time.Duration) {
	t.t = time.AfterFunc(timeOut, t.Trigger)
}

func (t *Timer) Stop() {
	t.t.Stop()
}

func (t *Timer) Trigger() {
	t.requestHalt(nil)
}
