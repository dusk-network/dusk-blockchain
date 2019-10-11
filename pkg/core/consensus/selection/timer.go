package selection

import (
	"time"
)

type timer struct {
	s *selector
	t *time.Timer
}

func (t *timer) start(timeOut time.Duration) {
	t.t = time.NewTimer(timeOut)
}

func (t *timer) stop() {
	t.t.Stop()
}

func (t *timer) trigger() {
	t.s.publishBestEvent()
}
