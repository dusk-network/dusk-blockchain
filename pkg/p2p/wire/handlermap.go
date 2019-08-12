package wire

import (
	"math/rand"
	"sync"
)

type idHandler struct {
	id uint32
	Handler
}

type handlerMap struct {
	lock     sync.RWMutex
	handlers map[string][]idHandler
}

func newHandlerMap() *handlerMap {
	return &handlerMap{
		handlers: make(map[string][]idHandler),
	}
}

// Store a Handler into an ordered slice stored at a key
func (h *handlerMap) Store(key string, value Handler) uint32 {
	id := rand.Uint32()
	h.lock.Lock()
	h.handlers[key] = append(h.handlers[key], idHandler{id, value})
	h.lock.Unlock()
	return id
}

// Load a copy of the handlers stored for a given key
func (h *handlerMap) Load(key string) []Handler {
	h.lock.RLock()
	handlers, _ := h.handlers[key]
	h.lock.RUnlock()

	hs := make([]Handler, len(handlers))
	for i, h := range handlers {
		hs[i] = h.Handler
	}
	return hs
}

// Delete a handler using the uint32 key returned during the Store operation. Return wether the item was found or otherwise
func (h *handlerMap) Delete(key string, id uint32) bool {
	found := false
	h.lock.Lock()
	handlers := h.handlers[key]
	for i, handler := range handlers {
		if handler.id == id {
			handler.Close()
			h.handlers[key] = append(
				h.handlers[key][:i],
				h.handlers[key][i+1:]...,
			)
			found = true
			break
		}
	}
	h.lock.Unlock()
	return found
}
