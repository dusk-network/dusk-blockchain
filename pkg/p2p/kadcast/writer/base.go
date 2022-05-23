// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package writer

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

var log = logrus.WithFields(logrus.Fields{"process": "kadcast"})

const (
	// MaxWriterQueueSize max number of messages queued for broadcasting.
	MaxWriterQueueSize = 1000
)

// Base is base impl of a kadcast writer.
type Base struct {
	subscriber     eventbus.Subscriber
	gossip         *protocol.Gossip
	subscriptionID uint32
	client         rusk.NetworkClient
	ctx            context.Context

	topic topics.Topic
}

// Send is a wrapper of rusk.NetworkClient Send method.
func (b *Base) Send(data []byte, addr string) error {
	// create the message
	blob := bytes.NewBuffer(data)

	// Make the message unique so it is not fitered out by kadcast cache.
	e, _ := crypto.RandEntropy(64)
	reserved := binary.LittleEndian.Uint64(e)

	if err := b.gossip.ProcessWithReserved(blob, reserved); err != nil {
		return err
	}

	// extract destination address
	// prepare message
	m := &rusk.SendMessage{
		TargetAddress: addr,
		Message:       blob.Bytes(),
	}

	md := metadata.New(map[string]string{"x-rusk-version": config.RuskVersion})
	ctxWithVersion := metadata.NewOutgoingContext(b.ctx, md)

	// send message
	if _, err := b.client.Send(ctxWithVersion, m); err != nil {
		log.WithError(err).Warn("failed to send message")
		return err
	}

	return nil
}

// Close unsubscribes.
func (b *Base) Close() error {
	b.subscriber.Unsubscribe(b.topic, b.subscriptionID)
	return nil
}
