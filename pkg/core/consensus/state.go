package consensus

import (
	"strconv"
	"sync"
)

var empty struct{}

type (
	// SyncState is an implementation of State which can be shared by multiple processes.
	// It also notifies subscribers of changes in the state's step.
	SyncState struct {
		Lock  sync.RWMutex
		round uint64
		step  uint8
	}

	// AsyncState is a representation of the consensus state at any given point in time.
	// Can be used to 'date' messages that are passed between consensus components.
	AsyncState struct {
		Round uint64
		Step  uint8
	}
)

// NewState returns an initialized SyncState.
func NewState() *SyncState {
	return &SyncState{
		round: 0,
		step:  1,
	}
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
	defer s.Lock.Unlock()
	s.step++
}
