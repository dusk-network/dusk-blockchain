// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package notifications

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type wsClient struct {
	conn wsConn
	// data to be sent as a websocket.TextMessage frame
	// closing msgChan terminates wsClient loop
	msgChan chan []byte
	id      string

	closed int32
}

func (c *wsClient) writeLoop() {
	// Teardown procedure for wsClient
	defer func() {
		// rfc6455#section-5.3
		// The closing handshake is intended to complement the TCP closing
		// handshake (FIN/ACK), on the basis that the TCP closing handshake is not
		// always reliable end-to-end, especially in the presence of intercepting
		// proxies and other intermediaries.
		_ = c.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second))

		err := c.conn.Close()
		if err != nil {
			log.WithError(err).
				WithField("conn_addr", c.id).
				Error("could not close connection")
		} else {
			log.WithField("conn_addr", c.id).Trace("close websocket client")
		}
	}()

	for msg := range c.msgChan {
		log.WithField("conn_addr", c.id).
			WithField("msg_len", len(msg)).Trace("send ws message")

		if err := c.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
			log.WithField("client_id", c.id).
				WithError(err).
				Warn("could not set write deadline")
			break
		}

		// NB: Use websocket.BinaryMessage + Compression if reducing frame size is a thing
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.WithField("conn_addr", c.id).
				WithError(err).Trace("close websocket client")
			// Instead of using a websocket.PingMessage to check client is alive,
			// we rely here on message sending as it's on regular base. If it fails,
			// the client is removed from the list of active clients
			break
		}
	}

	atomic.AddInt32(&c.closed, 1)
}

func (c *wsClient) readLoop() {
	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			break
		}
	}
}

func (c *wsClient) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}
