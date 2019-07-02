package generation

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
)

type seeder struct {
	lock  sync.RWMutex
	round uint64
	seed  []byte
	keys  user.Keys
}

func (s *seeder) GenerateSeed(prevSeed []byte, round uint64) ([]byte, error) {
	s.lock.Lock()
	seed, err := bls.Sign(s.keys.BLSSecretKey, s.keys.BLSPubKey, prevSeed)
	if err != nil {
		return nil, err
	}
	compSeed := seed.Compress()
	s.seed = compSeed
	s.round = round
	s.lock.Unlock()
	return compSeed, nil
}

func (s *seeder) LatestSeed() []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.seed
}

func (s *seeder) isFresh(seed []byte) bool {
	return bytes.Equal(s.seed, seed)
}

func (s *seeder) Round() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.round
}
