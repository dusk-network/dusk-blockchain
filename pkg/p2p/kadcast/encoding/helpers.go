// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"errors"

	"golang.org/x/crypto/blake2b"
)

// verifyIDNonce performs blake2b Sum256 over concatenated id and nonce to ensure last
// byte is zero.
func verifyIDNonce(id [16]byte, nonce []byte) error {
	if len(nonce) != 4 {
		return errors.New("invalid nonce length")
	}

	data := make([]byte, 0)
	data = append(data, id[:]...)
	data = append(data, nonce[:]...)

	hash := blake2b.Sum256(data)
	if (hash[31]) == 0 {
		return nil
	}

	return errors.New("id and Nonce are not valid parameters")
}

// MsgTypeToString translates wire message into string name.
func MsgTypeToString(t byte) (string, error) {
	switch t {
	case PingMsg:
		return "PING", nil
	case PongMsg:
		return "PONG", nil
	case FindNodesMsg:
		return "FIND_NODES", nil
	case NodesMsg:
		return "NODES", nil
	case BroadcastMsg:
		return "BROADCAST", nil
	}

	return "UNKNOWN", errors.New("not supported")
}

// ComputeNonce receives the user's `Peer` ID and computes the
// ID nonce in order to be able to join the network.
//
// This operation is basically a PoW algorithm that ensures
// that Sybil attacks are more costly.
func ComputeNonce(id []byte) uint32 {
	var nonce uint32 = 0

	var hash [32]byte
	var nonceBytes [NonceLen]byte
	var data [20]byte

	for {
		byteOrder.PutUint32(nonceBytes[:], nonce)

		copy(data[0:16], id[0:16])
		copy(data[16:20], nonceBytes[0:4])

		hash = blake2b.Sum256(data[:])
		if (hash[31]) == 0 {
			return nonce
		}

		// reset buffer
		for i := 0; i < len(nonceBytes); i++ {
			nonceBytes[i] = 0
		}

		nonce++
	}
}
