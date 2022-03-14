// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	g := protocol.NewGossip()

	m := bytes.NewBufferString("pippo")

	if !assert.NoError(t, g.Process(m)) {
		assert.FailNow(t, "error in processing buffer")
	}

	length, err := protocol.ReadFrame(m)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "error in reading frame")
	}

	msg := make([]byte, length)
	if _, err := m.Read(msg); err != nil {
		assert.FailNow(t, fmt.Sprintf("error in reading the message with length %d", length))
	}

	buf := new(bytes.Buffer)
	_, _ = buf.Write([]byte("pippo"))
	// First 20 bytes of `msg` are the version, checksum, and reserved bytes
	assert.Equal(t, buf.Bytes(), msg[(8+12):])
}

func TestUnpackLength(t *testing.T) {
	test := "pippo"
	b := bytes.NewBufferString(test)

	g := protocol.NewGossip()
	assert.NoError(t, g.Process(b))

	length, err := g.UnpackLength(b)
	assert.NoError(t, err)

	// Checksum was added, so we remove 4 from int(length)
	assert.Equal(t, len(test), int(length)-4)
}
