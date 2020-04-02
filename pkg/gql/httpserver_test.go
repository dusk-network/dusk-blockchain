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
)

func TestWebsocketEndpoint(t *testing.T) {

	// Set up HTTP server with notifications enabled
	// config
	s, eb, err := setupServer(t, "127.0.0.1:22222")
	if err != nil {
		t.Error(err)
	}
	defer s.Stop()

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Set up a websocket client
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:22222", Path: "/ws"}
	c, _, err := websocket.DefaultDialer.DialContext(dialCtx, u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = c.Close()
	}()

	response := make(chan string)
	go func() {
		_, msg, err := c.ReadMessage()
		if err != nil {
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

	blk := helper.RandomBlock(t, uint64(0), 4)
	hash, _ := blk.CalculateHash()
	blk.Header.Hash = hash
	msg := message.New(topics.AcceptedBlock, *blk)
	eb.Publish(topics.AcceptedBlock, msg)

	message := <-response

	t.Logf("Message size %d", len(message))

	expMsg, err := notifications.MarshalBlockMsg(*blk)
	if err != nil {
		t.Errorf("marshaling failed")
	}

	if message == "no response" {
		t.Fatalf("no response received")
	}

	if expMsg != message {
		t.Errorf("malformed message received")
	}
}

func setupServer(t *testing.T, addr string) (*Server, *eventbus.EventBus, error) {
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
		t.Fatal(err)
	}

	err = s.Start()

	return s, eb, err
}
