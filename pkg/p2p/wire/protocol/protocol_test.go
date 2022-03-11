// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol_test

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
)

var magicTest = []struct {
	magic   protocol.Magic
	payload []byte
}{
	{protocol.TestNet, []byte("pippo")},
	{protocol.DevNet, []byte("paperino")},
	{protocol.TestNet, []byte("pluto")},
	{protocol.MainNet, []byte("bruto")},
}

func TestExtract(t *testing.T) {
	var buf bytes.Buffer

	for _, tt := range magicTest {
		buf = tt.magic.ToBuffer()
		buf.ReadFrom(bytes.NewBuffer(tt.payload))
		magic, version, err := protocol.Extract(&buf)
		assert.Equal(t, magic, tt.magic)
		assert.NoError(t, err)
		assert.Equal(t, tt.payload, buf.Bytes())
		assert.Equal(t, version, protocol.CurrentProtocolVersion)
	}
}

// Ensure that a simple `Read` call from a net.Conn can result in a short read, and
// that use of `io.ReadFull` is preferred.
func TestShortRead(t *testing.T) {
	buffer := make([]byte, 4)
	pw, pr := net.Pipe()

	go delayedWrite(pw)

	if _, err := pr.Read(buffer); err != nil {
		t.Fatal(err)
	}

	// Last two bytes should not have been read, as the read would've finished early.
	assert.Equal(t, []byte{1, 2, 0, 0}, buffer)

	// Try again with `io.ReadFull`
	buffer = make([]byte, 4)
	pw, pr = net.Pipe()

	go delayedWrite(pw)

	if _, err := io.ReadFull(pr, buffer); err != nil {
		t.Fatal(err)
	}

	// We should now have the full four bytes
	assert.Equal(t, []byte{1, 2, 3, 4}, buffer)
}

func delayedWrite(c net.Conn) {
	// Send first two bytes
	if _, err := c.Write([]byte{1, 2}); err != nil {
		panic(err)
	}

	// Wait a bit, and write the other two
	time.Sleep(100 * time.Millisecond)

	if _, err := c.Write([]byte{3, 4}); err != nil {
		panic(err)
	}
}
