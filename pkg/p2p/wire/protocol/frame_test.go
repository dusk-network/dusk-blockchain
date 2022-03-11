// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

func TestWriteReadFrame(t *testing.T) {
	b := bytes.NewBufferString("pippo")
	digest := sha3.Sum256(b.Bytes())
	WriteFrame(b, TestNet, digest[0:checksum.Length])

	length, _ := ReadFrame(b)
	buf := make([]byte, length)
	b.Read(buf)

	// Remove magic, checksum, version, and reserved bytes
	buf = buf[(16 + 8):]

	assert.Equal(t, "pippo", string(buf))
}
