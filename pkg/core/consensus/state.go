package consensus

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type (
	State interface {
		fmt.Stringer
		Round() uint64
		Step() uint8
		Update(uint64)
		IncrementStep()
		Cmp(round uint64, step uint8) int
	}

	SyncState struct {
		Lock  sync.RWMutex
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
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.round
}

func (s *SyncState) Step() uint8 {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.step
}

func (s *SyncState) String() string {
	return "round: " + strconv.Itoa(int(s.Round())) +
		" / step: " + strconv.Itoa(int(s.Step()))
}

func (s *SyncState) Update(round uint64) {
	s.Lock.Lock()
	s.round = round
	s.step = 1
	s.Lock.Unlock()
}

func (s *SyncState) IncrementStep() {
	s.Lock.Lock()
	s.step++
	s.Lock.Unlock()
}

// Cmp returns negative number if the SyncState is in the future, 0 if they are the same and positive if the SyncState is in the past
func (s *SyncState) Cmp(round uint64, step uint8) int {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	cmp := s.round - round
	if cmp == 0 {
		return int(s.step) - int(step)
	}

	return int(cmp)
}
