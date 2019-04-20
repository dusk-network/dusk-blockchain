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
		SubscribeStep() *StepSubscriber
		Update(uint64)
		IncrementStep()
		Cmp(round uint64, step uint8) (int, int)
	}

	SyncState struct {
		Lock            sync.RWMutex
		round           uint64
		step            uint8
		stepSubscribers []*StepSubscriber
	}

	AsyncState struct {
		Round uint64
		Step  uint8
	}

	StepSubscriber struct {
		StateChan chan struct{}
		id        uint32
	}

	Timer struct {
		Timeout     time.Duration
		TimeoutChan chan struct{}
	}
)

func NewState() *SyncState {
	return &SyncState{
		round:           0,
		step:            1,
		stepSubscribers: make([]*StepSubscriber, 0),
	}
}

func (s *SyncState) SubscribeStep() *StepSubscriber {
	sub := &StepSubscriber{
		id:        rand.Uint32(),
		StateChan: make(chan struct{}, 1),
	}

	s.stepSubscribers = append(s.stepSubscribers, sub)
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
	for _, sub := range s.stepSubscribers {
		sub.StateChan <- empty
	}
}

// Cmp returns negative number if the SyncState is in the future, 0 if they are the same and positive if the SyncState is in the past
func (s *SyncState) Cmp(round uint64, step uint8) (int, int) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	roundDiff := int(s.round) - int(round)
	stepDiff := int(s.step) - int(step)
	return roundDiff, stepDiff
}
