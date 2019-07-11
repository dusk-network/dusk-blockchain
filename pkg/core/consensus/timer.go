package consensus

import (
	"sync"
	"time"
)

type Timer struct {
	lock        sync.RWMutex
	timeOut     time.Duration
	baseTimeOut time.Duration
	TimeOutChan chan struct{}
}

func NewTimer(timeOut time.Duration, timeOutChan chan struct{}) *Timer {
	return &Timer{
		timeOut:     timeOut,
		baseTimeOut: timeOut,
		TimeOutChan: timeOutChan,
	}
}

func (t *Timer) IncreaseTimeOut() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.timeOut < t.baseTimeOut*2^10 {
		t.timeOut = t.timeOut * 2
	}
}

func (t *Timer) ResetTimeOut() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.timeOut = t.baseTimeOut
}

func (t *Timer) TimeOut() time.Duration {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.timeOut
}
