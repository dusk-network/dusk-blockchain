// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package notifications

import (
	"container/list"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/sirupsen/logrus"
)

const (
	// Write deadline. After a write has timed out, the websocket state is
	// corrupt and all future writes will return an error.
	writeDeadline = 3 * time.Second

	maxTxsPerMsg = 15
)

var log = logrus.WithField("process", "consensus").
	WithField("actor", "broker")

// Broker is a pub/sub broker that keeps updated all subscribers (websocket
// connections) with latest block accepted published by node layer.
//
// IMPL Notes:
// Broker is implemented in a non-blocking manner. That means it should not be
// blocked on mutex-lock, chan push or any I/O operation.
// It should not expose anything than a channel.
type Broker struct {
	id uint

	// Active clients subscribed for updates.
	clients *list.List
	// Max number of clients per a broker instance.
	maxClientsCount uint

	// ConnectionChan is a shared queue to buffer incoming websocket connections
	// closing connChan will terminate the broker.
	ConnectionChan chan wsConn

	// Events.
	eventBus          eventbus.Broker
	acceptedBlockChan chan block.Block
	acceptedBlockID   uint32
}

// NewBroker creates a new Broker instance.
func NewBroker(id uint, eventBus eventbus.Broker, maxClientsCount uint, connChan chan wsConn) *Broker {
	b := new(Broker)
	b.eventBus = eventBus
	b.ConnectionChan = connChan
	b.acceptedBlockChan, b.acceptedBlockID = consensus.InitAcceptedBlockUpdate(eventBus)
	b.clients = list.New()
	b.maxClientsCount = maxClientsCount
	b.id = id
	return b
}

// Run represents the main loop of the Broker, where block data get piped to
// incoming connections. Connections are put in Idle state after 30 seconds of
// inactivity.
func (b *Broker) Run() {
	// Teardown procedure for the Broker.
	defer func() {
		// It's advisable to reset the broker state entirely.
		// This would allow later to restart a broker without any leaks.
		log.WithField("id", b.id).Info("closing")

		// Unsubscribe from all eventBus events.
		b.eventBus.Unsubscribe(topics.AcceptedBlock, b.acceptedBlockID)

		// Terminate all clients goroutines.
		for e := b.clients.Front(); e != nil; e = e.Next() {
			c := e.Value.(*wsClient)
			close(c.msgChan)
		}

		// reset clients list
		b.clients.Init()
	}()

	for {
		// Any of the handlers must be capable of recovering from panic
		select {
		// new client connection from webserver
		case conn, isOpen := <-b.ConnectionChan:
			if !isOpen {
				// Terminate the broker when the shared connChan is closed
				return
			}

			b.handleConn(conn)
		// new accepted block from node
		case blk := <-b.acceptedBlockChan:
			b.handleBlock(blk)
		case <-time.After(30 * time.Second):
			b.handleIdle()
		}
	}
}

// handleBlock handles the topics.AcceptedBlock event emitted from node layer.
// It packs a json from the block and broadcast it to all active clients.
func (b *Broker) handleBlock(blk block.Block) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("handleBlock recovered from err: %v", r)
		}
	}()

	b.reap()

	msg, err := MarshalBlockMsg(blk)
	if err != nil {
		log.Errorf("encoding err: %v", err)
	}

	b.broadcastMessage(msg)
}

// handleConn handles a new websocket conn pushed from webserver layer It stores
// the conn to list of active clients.
func (b *Broker) handleConn(conn wsConn) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("handleConn recovered from err: %v", r)
		}
	}()

	log.WithField("conn", conn.RemoteAddr().String()).Tracef("handle connection")

	b.reap()

	if b.clients.Len() >= int(b.maxClientsCount) {
		log.WithField("clients_num", b.clients.Len()).Warn("too many connections")
		// Discarding the connection
		_ = conn.Close()
		return
	}

	c := &wsClient{
		conn:    conn,
		msgChan: make(chan []byte, 100),
		id:      conn.RemoteAddr().String(),
	}

	_ = b.clients.PushBack(c)

	// Start a writer-goroutine dedicated for websocket conn. All messages to a
	// websocket-conn are sent via this writer-goroutine only. Whereas the
	// broker subscribes for the EventBus events, ws client subscribers only for
	// the data to be sent to the websocket-conn. Thus, eventBus event data and
	// data processing is not duplicated amongst all websocket connections
	// writers.
	go c.writeLoop()

	// Although, no message reading is necessary, we need to drain TCP receive buffer.
	go c.readLoop()
}

func (b *Broker) handleIdle() {
	log.
		WithField("id", b.id).
		WithField("clients_num", b.clients.Len()).
		Debug("onidle")
}

// broadcastMessage propagates data to all active clients.
func (b *Broker) broadcastMessage(data string) {
	if len(data) == 0 || b.clients.Len() == 0 {
		return
	}

	log.WithField("id", b.id).WithField("clients_num", b.clients.Len()).
		Debug("notify ws clients")

	log.WithField("body", data).Trace("broadcasted message")

	for e := b.clients.Front(); e != nil; e = e.Next() {
		c := e.Value.(*wsClient)
		c.msgChan <- []byte(data)
	}
}

// reap cleans up clients list from inactive/closed connections.
func (b *Broker) reap() {
	for e := b.clients.Front(); e != nil; {
		c := e.Value.(*wsClient)

		if c.IsClosed() {
			closedElm := e
			e = e.Next()

			b.clients.Remove(closedElm)
		} else {
			e = e.Next()
		}
	}
}
