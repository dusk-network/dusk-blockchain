package hashset

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"

var _empty = new(struct{})

type Set struct {
	entries map[string]*struct{}
}

func New() *Set {
	return &Set{
		make(map[string]*struct{}),
	}
}

// add the xxH64 hash of some data to the set. Returns false if the entry was already there
func (s *Set) Has(data []byte) bool {
	hs := repr(data)
	return s != nil && s.has(hs)
}

func (s *Set) Add(data []byte) bool {
	hs := repr(data)
	_, found := s.entries[hs]
	s.entries[hs] = _empty
	return found
}

func (s *Set) has(k string) bool {
	_, found := s.entries[k]
	return found
}

func repr(data []byte) string {
	hashed, _ := hash.Xxhash(data)
	return string(hashed)
}
