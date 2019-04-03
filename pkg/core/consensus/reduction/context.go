package reduction

import (
	"bytes"
	"sync"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
)

type (
	consensusState interface {
		Round() uint64
		Step() uint8
		Update(uint64)
		IncrementStep()
		Cmp(round uint64, step uint8) int
	}

	state struct {
		lock  sync.RWMutex
		round uint64
		step  uint8
	}

	timer struct {
		timeout     time.Duration
		timeoutChan chan interface{}
	}

	context struct {
		handler           handler
		committee         committee.Committee
		state             consensusState
		reductionVoteChan chan *bytes.Buffer
		agreementVoteChan chan *bytes.Buffer
		timer             *timer
	}
)

func (s *state) Round() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.round
}

func (s *state) Step() uint8 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.step
}

func (s *state) Update(round uint64) {
	s.lock.Lock()
	s.round = round
	s.step = 1
	s.lock.Unlock()
}

func (s *state) IncrementStep() {
	s.lock.Lock()
	s.step++
	s.lock.Unlock()
}

// Cmp returns negative number if the state is in the future, 0 if they are the same and positive if the state is in the past
func (s *state) Cmp(round uint64, step uint8) int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	cmp := s.round - round
	if cmp == 0 {
		return int(s.step - step)
	}

	return int(cmp)
}

func newCtx(handler handler, committee committee.Committee, timeout time.Duration) *context {
	return &context{
		handler:           handler,
		committee:         committee,
		state:             &state{sync.RWMutex{}, 0, 0},
		reductionVoteChan: make(chan *bytes.Buffer, 1),
		agreementVoteChan: make(chan *bytes.Buffer, 1),
		timer: &timer{
			timeout:     timeout,
			timeoutChan: make(chan interface{}, 1),
		},
	}
}
