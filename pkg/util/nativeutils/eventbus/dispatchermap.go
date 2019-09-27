package eventbus

import (
	"math/rand"
	"sync"
)

type idDispatcher struct {
	id uint32
	Dispatcher
}

type dispatcherMap struct {
	lock        sync.RWMutex
	dispatchers map[string][]idDispatcher
}

func newDispatcherMap() *dispatcherMap {
	return &dispatcherMap{
		dispatchers: make(map[string][]idDispatcher),
	}
}

// Store a Dispatcher into an ordered slice stored at a key
func (h *dispatcherMap) Store(key string, value Dispatcher) uint32 {
	id := rand.Uint32()
	h.lock.Lock()
	h.dispatchers[key] = append(h.dispatchers[key], idDispatcher{id, value})
	h.lock.Unlock()
	return id
}

// Load a copy of the dispatchers stored for a given key
func (h *dispatcherMap) Load(key string) []Dispatcher {
	h.lock.RLock()
	dispatchers, _ := h.dispatchers[key]
	h.lock.RUnlock()

	hs := make([]Dispatcher, len(dispatchers))
	for i, h := range dispatchers {
		hs[i] = h.Dispatcher
	}
	return hs
}

// Delete a dispatcher using the uint32 key returned during the Store operation. Return wether the item was found or otherwise
func (h *dispatcherMap) Delete(key string, id uint32) bool {
	found := false
	h.lock.Lock()
	dispatchers := h.dispatchers[key]
	for i, dispatcher := range dispatchers {
		if dispatcher.id == id {
			dispatcher.Close()
			h.dispatchers[key] = append(
				h.dispatchers[key][:i],
				h.dispatchers[key][i+1:]...,
			)
			found = true
			break
		}
	}
	h.lock.Unlock()
	return found
}
