// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package gql

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/gql/notifications"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/require"
)

func TestWebsocketEndpoint(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	assert := assert.New(t)

	// Set up HTTP server with notifications enabled
	// config
	s, eb, err := setupServer("127.0.0.1:22222")
	assert.NoError(err)
	defer s.Stop()

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Set up a websocket client
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:22222", Path: "/ws"}
	c, _, e := websocket.DefaultDialer.DialContext(dialCtx, u.String(), nil)
	assert.NoError(e)
	defer func() {
		_ = c.Close()
	}()

	response := make(chan string)
	go func() {
		_, msg, readErr := c.ReadMessage()
		if readErr != nil {
			return
		}
		response <- string(msg)
	}()

	// Unblock test if no response sent for N seconds
	go func() {
		time.Sleep(7 * time.Second)
		response <- "no response"
	}()

	// Simulate eventBus publishing a acceptedBlocks message

	time.Sleep(time.Second)

	blk := helper.RandomBlock(uint64(0), 4)
	hash, _ := blk.CalculateHash()
	blk.Header.Hash = hash
	msg := message.New(topics.AcceptedBlock, *blk)
	errList := eb.Publish(topics.AcceptedBlock, msg)
	assert.Empty(errList)

	message := <-response

	assert.Greater(len(message), 0)

	expMsg, err := notifications.MarshalBlockMsg(*blk)
	assert.NoError(err)
	assert.NotEqual("no response", expMsg)
	assert.NotEqual("malformed message received", expMsg)
}

func setupServer(addr string) (*Server, *eventbus.EventBus, error) {
	// Set up HTTP server with notifications enabled
	// config
	r := config.Registry{}
	r.Gql.Network = "tcp"
	r.Gql.Address = addr
	r.Gql.Enabled = true
	r.Gql.EnableTLS = false
	r.Gql.Notification.BrokersNum = 1
	r.Database.Driver = lite.DriverName
	r.General.Network = "testnet"
	config.Mock(&r)

	eb := eventbus.New()
	rpcBus := rpcbus.New()
	s, err := NewHTTPServer(eb, rpcBus)
	if err != nil {
		return nil, nil, err
	}

	return s, eb, s.Start()
}
