package eventbus

import (
	"math/rand"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type idListener struct {
	id uint32
	Listener
}

type listenerMap struct {
	lock      sync.RWMutex
	listeners map[topics.Topic][]idListener
}

func newListenerMap() *listenerMap {
	return &listenerMap{
		listeners: make(map[topics.Topic][]idListener),
	}
}

// Store a Listener into an ordered slice stored at a key
func (h *listenerMap) Store(key topics.Topic, value Listener) uint32 {
	id := rand.Uint32()
	h.lock.Lock()
	defer h.lock.Unlock()
	h.listeners[key] = append(h.listeners[key], idListener{id, value})
	return id
}

// Load a copy of the listeners stored for a given key
func (h *listenerMap) Load(key topics.Topic) []idListener {
	h.lock.RLock()
	defer h.lock.RUnlock()
	listeners := h.listeners[key]
	dup := make([]idListener, len(listeners))
	copy(dup, listeners)
	return dup
}

// Delete a listener using the uint32 key returned during the Store operation. Return wether the item was found or otherwise
func (h *listenerMap) Delete(key topics.Topic, id uint32) bool {
	found := false
	h.lock.Lock()
	defer h.lock.Unlock()
	listeners := h.listeners[key]
	for i, listener := range listeners {
		if listener.id == id {
			listener.Close()
			h.listeners[key] = append(
				h.listeners[key][:i],
				h.listeners[key][i+1:]...,
			)
			found = true
			break
		}
	}
	return found
}
