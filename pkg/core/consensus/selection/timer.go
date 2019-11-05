package selection

import (
	"time"
)

type timer struct {
	s *Selector
	t *time.Timer
}

func (t *timer) start(timeOut time.Duration) {
	t.t = time.AfterFunc(timeOut, t.trigger)
}

func (t *timer) stop() {
	if t.t != nil {
		t.t.Stop()
	}
}

func (t *timer) trigger() {
	t.s.publishBestEvent()
}
