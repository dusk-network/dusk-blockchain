package hashset

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"

var _empty = new(struct{})

type Set struct {
	entries map[string]*struct{}
}

func New() *Set {
	return &Set{
		entries: make(map[string]*struct{}),
	}
}

// Has returns true if the entry is found
func (s *Set) Has(data []byte) bool {
	hs := repr(data)
	return s != nil && s.has(hs)
}

// Add an entry to the current set. If the entry is already there, it returns true
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

func (s *Set) Size() int {
	return len(s.entries)
}

func repr(data []byte) string {
	hashed, _ := hash.Xxhash(data)
	return string(hashed)
}
