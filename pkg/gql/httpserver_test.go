// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package gql

import (
	"context"
	"crypto/tls"
	"net/url"
	"syscall"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/require"
)

const (
	// clientsNum number of ws clients to try to connect to Notification service.
	clientsNum = 2*1000 + 500

	// maxAllowedClients is max connections the service is configured to handle.
	maxAllowedClients = uint(2 * 500)
	brokersNum        = uint(2)

	// srvAddr Notification service address.
	srvAddr = "127.0.0.1:22222"

	// TestMultiClient can be executed over TLS, if certFile, keyFile are provided.
	enableTLS = false
	certFile  = "./example.crt"
	keyFile   = "./example.key"

	maxOpenFiles = 10 * 1000
)

// TestMultiClient ensures Notification service is capable of handling up to
// broker * clients connections concurrently.
func TestMultiClient(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)

	assert := assert.New(t)

	increaseFDLimit(maxOpenFiles)

	// Set up HTTP server with notifications enabled
	// config
	s, eb, err := createServer(srvAddr, brokersNum, maxAllowedClients/brokersNum, enableTLS)
	assert.NoError(err)

	defer s.Close()

	// Set up a list of ws clients
	resp := make(chan string, clientsNum)

	for i := 0; i < clientsNum; i++ {
		if err := createClient(srvAddr, resp, enableTLS); err != nil {
			assert.NoError(err)
		}
	}

	// Simulate eventBus publishing an acceptedBlocks message
	publishRandBlock(eb, assert)

	// Run a simple monitor to ensure a condition is satisfied
	respCount := uint(0)
	done := make(chan bool)

	go func(ch chan bool, c *uint) {
		for {
			<-resp
			*c++
			// Ensure that the count of ws connections that received a
			// Block Update Message is neither more nor less than
			// maxAllowedClients
			if *c == maxAllowedClients {
				ch <- true
				return
			}
		}
	}(done, &respCount)

	// Within up to 10 sec, we expect all ws clients have received a Block Update Message
	select {
	case <-done:
		break
	case <-time.After(10 * time.Second):
		assert.Failf("could not receive all responses", "number of responses", respCount)
	}
}

func createClient(addr string, resp chan string, enableTLS bool) error {
	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Set up a websocket (secure/insecure) client
	scheme := "ws"

	if enableTLS {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{}
		websocket.DefaultDialer.TLSClientConfig.InsecureSkipVerify = true

		scheme = "wss"
	}

	u := url.URL{Scheme: scheme, Host: addr, Path: "/" + scheme}

	c, _, err := websocket.DefaultDialer.DialContext(dialCtx, u.String(), nil)
	if err != nil {
		return err
	}

	go func() {
		for {
			_, msg, readErr := c.ReadMessage()
			if readErr != nil {
				c.Close()
				break
			}
			resp <- string(msg)
		}
	}()

	return nil
}

func createServer(addr string, brokerNum, clientsPerBroker uint, enableTLS bool) (*Server, *eventbus.EventBus, error) {
	// Set up HTTP server with notifications enabled
	// config
	r := config.Registry{}
	r.Gql.Network = "tcp"
	r.Gql.Address = addr
	r.Gql.Enabled = true
	r.Gql.EnableTLS = enableTLS
	r.Gql.CertFile = certFile
	r.Gql.KeyFile = keyFile
	r.Gql.Notification.BrokersNum = brokerNum
	r.Gql.Notification.ClientsPerBroker = clientsPerBroker
	r.Database.Driver = lite.DriverName
	r.General.Network = "testnet"
	config.Mock(&r)

	eb := eventbus.New()
	rpcBus := rpcbus.New()

	s, err := NewHTTPServer(eb, rpcBus)
	if err != nil {
		return nil, nil, err
	}

	return s, eb, s.Start(context.Background())
}

func publishRandBlock(eb *eventbus.EventBus, assert *assert.Assertions) {
	blk := helper.RandomBlock(uint64(0), 4)
	hash, _ := blk.CalculateHash()
	blk.Header.Hash = hash
	msg := message.New(topics.AcceptedBlock, *blk)

	errList := eb.Publish(topics.AcceptedBlock, msg)
	assert.Empty(errList)
}

func increaseFDLimit(l uint64) {
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		panic(err)
	}

	limit.Cur = l
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		panic(err)
	}
}
