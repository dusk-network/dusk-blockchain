package consensus

import "time"

type Timer struct {
	TimeOut     time.Duration
	baseTimeOut time.Duration
	TimeOutChan chan struct{}
}

func NewTimer(timeOut time.Duration, timeOutChan chan struct{}) *Timer {
	return &Timer{
		TimeOut:     timeOut,
		baseTimeOut: timeOut,
		TimeOutChan: timeOutChan,
	}
}

func (t *Timer) IncreaseTimeOut() {
	t.TimeOut = t.TimeOut * 2
}

func (t *Timer) ResetTimeOut() {
	t.TimeOut = t.baseTimeOut
}
