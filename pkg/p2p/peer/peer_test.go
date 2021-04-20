// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

var receiveFn = func(c net.Conn) {
	for {
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 1024)
		if _, err := c.Read(buf); err != nil {
			break
		}
	}
}

// Test the functionality of the peer.Reader through the ReadLoop.
func TestReader(t *testing.T) {
	g := protocol.NewGossip(protocol.TestNet)
	client, srv := net.Pipe()

	eb := eventbus.New()

	// Set up reader factory
	processor := NewMessageProcessor(eb)
	agreementChan := make(chan struct{}, 1)
	respFn := func(_ string, _ message.Message) ([]bytes.Buffer, error) {
		agreementChan <- struct{}{}
		return nil, nil
	}

	processor.Register(topics.Agreement, respFn)
	factory := NewReaderFactory(processor)

	responseChan := make(chan bytes.Buffer, 100)
	peerReader := factory.SpawnReader(srv, protocol.NewGossip(protocol.TestNet), responseChan)

	go peerReader.ReadLoop(context.Background())

	errChan := make(chan error, 1)
	go func(eChan chan error) {
		msg := makeAgreementGossip(10)

		buf, err := message.Marshal(msg)
		if err != nil {
			eChan <- err
		}

		if err := g.Process(&buf); err != nil {
			eChan <- err
		}

		_, _ = client.Write(buf.Bytes())
	}(errChan)

	// We should get the message through this channel
	select {
	case err := <-errChan:
		t.Fatal(err)
	case <-agreementChan:
	}
}

// Test the functionality of the peer.Writer through the use of the ring buffer.
func TestWriteRingBuffer(t *testing.T) {
	bus := eventbus.New()

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		_ = p.Conn.Close()
	}

	ev := makeAgreementGossip(10)
	msg := message.New(topics.Agreement, ev)

	for i := 0; i < 1000; i++ {
		errList := bus.Publish(topics.Gossip, msg)
		require.Empty(t, errList)
	}
}

// Test the functionality of the peer.Writer through the use of the outgoing message queue.
func TestWriteLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	g := protocol.NewGossip(protocol.TestNet)
	msg := makeAgreementGossip(10)

	buf, err := message.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	go func(g *protocol.Gossip) {
		responseChan := make(chan bytes.Buffer)

		writer := NewWriter(client, g, bus, 30*time.Millisecond)
		go writer.Serve(context.Background(), responseChan)

		bufCopy := buf
		responseChan <- bufCopy
	}(g)

	// Decode and remove magic
	length, err := g.UnpackLength(srv)
	assert.NoError(t, err)

	decoded := make([]byte, length)
	_, err = io.ReadFull(srv, decoded)
	assert.NoError(t, err)

	// Remove checksum
	decoded = decoded[4:]

	assert.Equal(t, decoded, (&buf).Bytes())
}

func BenchmarkWriter(t *testing.B) {
	bus := eventbus.New()

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		_ = p.Conn.Close()
	}

	msg := makeAgreementGossip(10)

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		errList := bus.Publish(topics.Gossip, msg)
		require.Empty(t, errList)
	}
}

//nolint:unparam
func makeAgreementGossip(keyAmount int) message.Message {
	p, keys := consensus.MockProvisioners(keyAmount)
	aggro := message.MockAgreement(make([]byte, 32), 1, 1, keys, p)
	return message.New(topics.Agreement, aggro)
}

func addPeer(bus *eventbus.EventBus, receiveFunc func(net.Conn)) *Writer {
	client, srv := net.Pipe()
	g := protocol.NewGossip(protocol.TestNet)
	pw := NewWriter(client, g, bus)

	go receiveFunc(srv)
	return pw
}
