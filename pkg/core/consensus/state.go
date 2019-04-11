package consensus

import (
	"sync"
	"time"
)

type (
	State interface {
		Round() uint64
		Step() uint8
		Update(uint64)
		IncrementStep()
		Cmp(round uint64, step uint8) int
	}

	SyncState struct {
		lock  sync.RWMutex
		round uint64
		step  uint8
	}

	Timer struct {
		Timeout     time.Duration
		TimeoutChan chan interface{}
	}
)

func NewState() *SyncState {
	return &SyncState{sync.RWMutex{}, 0, 1}
}

func (s *SyncState) Round() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.round
}

func (s *SyncState) Step() uint8 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.step
}

func (s *SyncState) Update(round uint64) {
	s.lock.Lock()
	s.round = round
	s.step = 1
	s.lock.Unlock()
}

func (s *SyncState) IncrementStep() {
	s.lock.Lock()
	s.step++
	s.lock.Unlock()
}

// Cmp returns negative number if the SyncState is in the future, 0 if they are the same and positive if the SyncState is in the past
func (s *SyncState) Cmp(round uint64, step uint8) int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	cmp := s.round - round
	if cmp == 0 {
		return int(s.step) - int(step)
	}

	return int(cmp)
}
