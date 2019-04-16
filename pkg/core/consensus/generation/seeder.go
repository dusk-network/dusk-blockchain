package generation

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

type seeder struct {
	sync.RWMutex
	round uint64
	seed  []byte
}

func (s *seeder) GenerateSeed(round uint64) []byte {
	// TODO: make an actual seed by signing the previous block seed
	seed, _ := crypto.RandEntropy(33)
	s.Lock()
	s.seed = seed
	s.round = round
	s.Unlock()
	return seed
}

func (s *seeder) LatestSeed() []byte {
	s.RLock()
	defer s.RUnlock()
	return s.seed
}

func (s *seeder) isFresh(seed []byte) bool {
	return bytes.Equal(s.seed, seed)
}

func (s *seeder) Round() uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.round
}
