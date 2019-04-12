package consensus

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var empty struct{}

type (
	State interface {
		fmt.Stringer
		Round() uint64
		Step() uint8
		SubscribeState() *StateSubscriber
		Update(uint64)
		IncrementStep()
		Cmp(round uint64, step uint8) int
	}

	AsyncState struct {
		Step  uint8
		Round uint64
	}

	SyncState struct {
		Lock             sync.RWMutex
		round            uint64
		step             uint8
		stateSubscribers []*StateSubscriber
	}

	StateSubscriber struct {
		StateChan chan struct{}
		id        uint32
	}

	Timer struct {
		Timeout     time.Duration
		TimeoutChan chan interface{}
	}
)

func (a *AsyncState) Cmp(round uint64, step uint8) int {
	cmp := a.Round - round
	if cmp == 0 {
		return int(a.Step) - int(step)
	}

	return int(cmp)
}

func (a *AsyncState) String() string {
	return "round: " + strconv.Itoa(int(a.Round)) +
		" / step: " + strconv.Itoa(int(a.Step))
}

func NewState() *SyncState {
	return &SyncState{
		round:            0,
		step:             1,
		stateSubscribers: make([]*StateSubscriber, 0),
	}
}

func (s *SyncState) SubscribeState() *StateSubscriber {
	sub := &StateSubscriber{
		id:        rand.Uint32(),
		StateChan: make(chan struct{}, 1),
	}

	s.stateSubscribers = append(s.stateSubscribers, sub)
	return sub
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
	for _, sub := range s.stateSubscribers {
		sub.StateChan <- empty
	}
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
