// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	"crypto/rand"
	"math/big"
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

// Store a Listener into an ordered slice stored at a key.
func (h *listenerMap) Store(key topics.Topic, value Listener) uint32 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(32))
	if err != nil {
		panic(err)
	}

	n := nBig.Int64()
	id := uint32(n)

	h.lock.Lock()
	h.listeners[key] = append(h.listeners[key], idListener{id, value})
	h.lock.Unlock()

	return id
}

// Load a copy of the listeners stored for a given key.
func (h *listenerMap) Load(key topics.Topic) []idListener {
	h.lock.RLock()
	defer h.lock.RUnlock()

	listeners := h.listeners[key]
	dup := make([]idListener, len(listeners))
	copy(dup, listeners)

	return dup
}

// Delete a listener using the uint32 key returned during the Store operation. Return wether the item was found or otherwise.
func (h *listenerMap) Delete(key topics.Topic, id uint32) bool {
	h.lock.Lock()

	found := false

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

	h.lock.Unlock()
	return found
}

func (h *listenerMap) Close() {
	h.lock.Lock()
	defer h.lock.Unlock()

	for _, listeners := range h.listeners {
		for _, listener := range listeners {
			listener.Close()
		}
	}

	h.listeners = nil
}
