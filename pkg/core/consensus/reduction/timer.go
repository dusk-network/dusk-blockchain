package reduction

import "time"

type timer struct {
	r *reducer
	t *time.Timer
}

func (t *timer) start(timeOut time.Duration) {
	t.t = time.NewTimer(timeOut)
}

func (t *timer) stop() {
	t.t.Stop()
}

func (t *timer) trigger() {
	t.r.resetStepVotes()
	t.r.stepper.RequestStepUpdate()
}
