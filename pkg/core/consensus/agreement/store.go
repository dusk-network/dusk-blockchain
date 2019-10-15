package agreement

import "sync"

type store struct {
	sync.RWMutex
	collected map[string][]Agreement
}

func newStore() *store {
	return &store{
		collected: make(map[string][]Agreement),
	}
}

// Put collects the Agreement and returns the number of agreement stored for a blockhash
func (s *store) Insert(blockHash string, a Agreement) int {
	s.Lock()
	defer s.Unlock()

	coll, found := s.collected[blockHash]
	if !found {
		s.collected[blockHash] = []Agreement{a}
		return 1
	}

	coll = append(coll, a)
	length := len(coll)
	s.collected[blockHash] = coll
	return length
}

func (s *store) Get(hash string) []Agreement {
	s.RLock()
	defer s.RUnlock()
	return s.collected[hash]
}
