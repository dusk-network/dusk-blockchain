// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

type storedAgreements []message.Agreement

func (s storedAgreements) Len() int {
	return len(s)
}

func (s storedAgreements) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s storedAgreements) Less(i, j int) bool {
	return s[i].Repr.Cmp(s[j].Repr) <= 0
}

func (s storedAgreements) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString("[\n")

	for i, aggro := range s {
		if i > 0 {
			_, _ = sb.WriteString("\n")
		}

		_, _ = sb.WriteString(aggro.String())
	}

	_, _ = sb.WriteString("\n")
	_, _ = sb.WriteString("]")
	return sb.String()
}

type store struct {
	sync.RWMutex
	collected map[uint8]storedAgreements
	createdAt int64
}

func newStore() *store {
	return &store{
		collected: make(map[uint8]storedAgreements),
		createdAt: time.Now().Unix(),
	}
}

func (s *store) String() string {
	var sb strings.Builder

	for k, v := range s.collected {
		_, _ = sb.WriteString(strconv.Itoa(int(k)))
		_, _ = sb.WriteString(": ")
		_, _ = sb.WriteString(v.String())
		_, _ = sb.WriteString("\n")
	}

	return sb.String()
}

func (s *store) Size() int {
	var i int

	for _, v := range s.collected {
		i += len(v)
	}

	return i
}

// Put collects the Agreement and returns the number of agreement stored for a blockhash.
func (s *store) Insert(a message.Agreement, weight int) int {
	s.Lock()
	defer s.Unlock()

	hdr := a.State()

	idx := s.find(a)
	if idx == -1 {
		agreements := make([]message.Agreement, weight)
		for i := range agreements {
			agreements[i] = a
		}

		s.collected[hdr.Step] = storedAgreements(agreements)
		return weight
	}

	stored := s.collected[hdr.Step]
	// if the Agreement is already in the store we do not add it
	if s.contains(idx, a) {
		return len(stored)
	}

	// efficient insertion with minimal element copy and no additional allocation
	// github.com/golang.go/wiki/SliceTricks
	for i := 0; i < weight; i++ {
		stored = append(stored, message.Agreement{})
		copy(stored[idx+1:], stored[idx:])

		stored[idx] = a
	}

	s.collected[hdr.Step] = stored
	return len(stored)
}

func (s *store) Get(step uint8) []message.Agreement {
	s.RLock()
	defer s.RUnlock()
	return s.collected[step]
}

func (s *store) Find(a message.Agreement) int {
	s.RLock()
	defer s.RUnlock()
	return s.find(a)
}

// Find returns the index of an Agreement in the stored collection or, if the Agreement has not been stored, the index at which it would be stored.
// In case no Agreement is stored for the blockHash specified, it returns -1.
func (s *store) find(a message.Agreement) int {
	hdr := a.State()

	stored := s.collected[hdr.Step]
	if stored == nil {
		return -1
	}

	return sort.Search(len(stored), func(i int) bool {
		return stored[i].Cmp(a) <= 0
	})
}

func (s *store) Contains(a message.Agreement) bool {
	s.RLock()
	defer s.RUnlock()

	idx := s.find(a)
	return s.contains(idx, a)
}

func (s *store) contains(idx int, a message.Agreement) bool {
	hdr := a.State()
	stored := s.collected[hdr.Step]

	if idx == -1 {
		return false
	}

	if idx < len(stored) && stored[idx].Equal(a) {
		return true
	}

	return false
}

func (s *store) Clear() {
	s.Lock()
	defer s.Unlock()

	for k := range s.collected {
		delete(s.collected, k)
	}
}

func (s *store) CreatedAt() int64 {
	s.RLock()
	defer s.RUnlock()

	return s.createdAt
}

func (s *store) Len() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.collected)
}
