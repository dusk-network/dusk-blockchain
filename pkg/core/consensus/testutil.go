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

type SimplePlayer struct {
	lock  sync.RWMutex
	step  uint8
	Round uint64
	State State
}

func NewSimplePlayer() *SimplePlayer {
	return &SimplePlayer{
		step:  1,
		Round: 1,
	}
}

// Play upticks the step
func (s *SimplePlayer) Play() uint8 {
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
	s.State = PAUSED
}

// Resume as specified by the EventPlayer interface
func (s *SimplePlayer) Resume(id uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.State = RUNNING
}
