// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package wire_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
)

func TestAddTopic(t *testing.T) {
	buf := bytes.NewBufferString("This is a test")
	topic := topics.Gossip
	err := topics.Prepend(buf, topic)
	assert.NoError(t, err)
	assert.Equal(t, []byte{byte(topics.Gossip), 0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x74, 0x65, 0x73, 0x74}, buf.Bytes())
}
