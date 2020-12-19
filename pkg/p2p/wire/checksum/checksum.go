// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package checksum

import (
	"bytes"
	"errors"

	"golang.org/x/crypto/blake2b"
)

//nolint:golint
const Length = 4

// Generate a blake2b Sum256 digest
func Generate(m []byte) []byte {
	digest := blake2b.Sum256(m)
	return digest[:Length]
}

// Extract message and checksum from  buffer
func Extract(m []byte) ([]byte, []byte, error) {
	// First 4 bytes are the checksum
	if len(m) < Length {
		return nil, nil, errors.New("invalid buffer size")
	}

	checksum := m[:Length]
	message := m[Length:]
	return message, checksum, nil
}

// Verify blake2b Sum256 digest
func Verify(m []byte, checksum []byte) bool {
	digest := blake2b.Sum256(m)
	return bytes.Equal(checksum, digest[:Length])
}
