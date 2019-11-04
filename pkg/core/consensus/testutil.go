package consensus

import "sync"

// State indicates the status of the EventPlayer
type State uint8

const (
	// PAUSED player
	PAUSED State = iota
	// RUNNING player
	RUNNING
)

// SimplePlayer is used within tests to simulate the behaviour of the
// consensus.EventPlayer
type SimplePlayer struct {
	lock  sync.RWMutex
	step  uint8
	Round uint64
	state State
}

// NewSimplePlayer creates a SimplePlayer
func NewSimplePlayer() *SimplePlayer {
	return &SimplePlayer{
		step:  1,
		Round: 1,
	}
}

// Play upticks the step
func (s *SimplePlayer) Play(uint32) uint8 {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.step++
	return s.step
}

// Step guards the step with a lock
func (s *SimplePlayer) Step() uint8 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.step
}

// Pause as specified by the EventPlayer interface
func (s *SimplePlayer) Pause(id uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.state = PAUSED
}

// Resume as specified by the EventPlayer interface
func (s *SimplePlayer) Resume(id uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.state = RUNNING
}

// State s a threadsafe method to return whether the player is paused or not
func (s *SimplePlayer) State() State {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.state
}
