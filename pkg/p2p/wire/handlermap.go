package wire

import "sync"

type handlerMap struct {
	lock     sync.RWMutex
	handlers map[string][]Handler
}

func newHandlerMap() *handlerMap {
	return &handlerMap{
		handlers: make(map[string][]Handler),
	}
}

func (h *handlerMap) Store(key string, value Handler) {
	h.lock.Lock()
	h.handlers[key] = append(h.handlers[key], value)
	h.lock.Unlock()
}

func (h *handlerMap) Load(key string) []Handler {
	h.lock.RLock()
	handlers, _ := h.handlers[key]
	h.lock.RUnlock()
	return handlers
}

func (h *handlerMap) Delete(key string, id uint32) {
	h.lock.Lock()
	handlers := h.handlers[key]
	for i, handler := range handlers {
		if handler.ID() == id {
			h.handlers[key] = append(h.handlers[key][:i],
				h.handlers[key][i+1:]...)
		}
	}
	h.lock.Unlock()
}
