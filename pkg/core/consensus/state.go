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
	// State comprises the methods to maintain a state of the consensus.
	State interface {
		fmt.Stringer
		Round() uint64
		Step() uint8
		SubscribeStep() *StepSubscriber
		Update(uint64)
		IncrementStep()
		Cmp(round uint64, step uint8) (int, int)
	}

	// SyncState is an implementation of State which can be shared by multiple processes.
	// It also notifies subscribers of changes in the state's step.
	SyncState struct {
		Lock            sync.RWMutex
		round           uint64
		step            uint8
		stepSubscribers []*StepSubscriber
	}

	// AsyncState is a representation of the consensus state at any given point in time.
	// Can be used to 'date' messages that are passed between consensus components.
	AsyncState struct {
		Round uint64
		Step  uint8
	}

	// StepSubscriber notifies its owner of a change in the state's step.
	StepSubscriber struct {
		StateChan chan struct{}
		id        uint32
	}

	// Timer is used to regulate the consensus components which work with a timeout.
	Timer struct {
		Timeout     time.Duration
		TimeoutChan chan struct{}
	}
)

// NewState returns an initialized SyncState.
func NewState() *SyncState {
	return &SyncState{
		round:           0,
		step:            1,
		stepSubscribers: make([]*StepSubscriber, 0),
	}
}

// SubscribeStep returns a StepSubscriber which notifies its owner of a change in
// the state's step.
func (s *SyncState) SubscribeStep() *StepSubscriber {
	sub := &StepSubscriber{
		id:        rand.Uint32(),
		StateChan: make(chan struct{}, 1),
	}

	s.stepSubscribers = append(s.stepSubscribers, sub)
	return sub
}

// Round returns the round that the SyncState is on.
func (s *SyncState) Round() uint64 {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.round
}

// Step returns the step that the SyncState is on.
func (s *SyncState) Step() uint8 {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.step
}

func (s *SyncState) String() string {
	return "round: " + strconv.Itoa(int(s.Round())) +
		" / step: " + strconv.Itoa(int(s.Step()))
}

// Update the round of the SyncState.
func (s *SyncState) Update(round uint64) {
	s.Lock.Lock()
	s.round = round
	s.step = 1
	s.Lock.Unlock()
}

// IncrementStep increments the SyncState step by 1. It also notifies any subscribers
// of the state change.
func (s *SyncState) IncrementStep() {
	s.Lock.Lock()
	s.step++
	s.Lock.Unlock()
	for _, sub := range s.stepSubscribers {
		sub.StateChan <- empty
	}
}

// Cmp returns negative number if the SyncState is in the future, 0 if they are the
// same and positive if the SyncState is in the past.
func (s *SyncState) Cmp(round uint64, step uint8) (int, int) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	roundDiff := int(s.round) - int(round)
	stepDiff := int(s.step) - int(step)
	return roundDiff, stepDiff
}
