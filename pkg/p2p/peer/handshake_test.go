// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"net"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/stretchr/testify/require"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

func TestHandshake(t *testing.T) {
	// setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	eb := eventbus.New()

	processor := NewMessageProcessor(eb)
	factory := NewReaderFactory(processor)

	client, srv := net.Pipe()

	g := protocol.NewGossip(protocol.TestNet)
	pConn := NewConnection(client, g)
	pw := NewWriter(pConn, eb)

	go func() {
		pConn := NewConnection(srv, protocol.NewGossip(protocol.TestNet))

		peerReader := factory.SpawnReader(pConn)
		if err := peerReader.Accept(protocol.FullNode); err != nil {
			panic(err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	defer func() {
		_ = pw.Conn.Close()
	}()

	if err := pw.Handshake(protocol.FullNode); err != nil {
		t.Fatal(err)
	}
}
